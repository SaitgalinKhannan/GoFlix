use rfd::FileDialog;
use serde::Serialize;
use std::{
    env, fs,
    path::{Path, PathBuf},
    process::Command,
};

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct DataDirectory {
    path: String,
    project_dir: String,
}

fn project_dir() -> Result<PathBuf, String> {
    if let Some(path) = env::var_os("GOFLIX_PROJECT_DIR") {
        let path = PathBuf::from(path)
            .canonicalize()
            .map_err(|err| format!("cannot access GOFLIX_PROJECT_DIR: {err}"))?;
        if path.join("docker-compose.yml").is_file() {
            return Ok(path);
        }
        return Err(format!(
            "GOFLIX_PROJECT_DIR ({}) does not contain docker-compose.yml",
            path.display()
        ));
    }

    let mut path = env::current_dir()
        .map_err(|err| format!("cannot determine current directory: {err}"))?
        .canonicalize()
        .map_err(|err| format!("cannot access current directory: {err}"))?;
    loop {
        if path.join("docker-compose.yml").is_file() {
            return Ok(path);
        }
        if !path.pop() {
            break;
        }
    }
    Err("could not find the GoFlix project directory; set GOFLIX_PROJECT_DIR".to_string())
}

fn env_path(project_dir: &Path) -> PathBuf {
    project_dir.join(".env")
}

fn unquote_env_value(value: &str) -> String {
    let value = value.trim();
    if value.len() >= 2
        && ((value.starts_with('"') && value.ends_with('"'))
            || (value.starts_with('\'') && value.ends_with('\'')))
    {
        value[1..value.len() - 1]
            .replace("\\\"", "\"")
            .replace("\\\\", "\\")
    } else {
        value.to_string()
    }
}

fn configured_data_dir(project_dir: &Path) -> Result<PathBuf, String> {
    let env_file = env_path(project_dir);
    let contents = fs::read_to_string(&env_file).unwrap_or_default();
    let configured = contents
        .lines()
        .find_map(|line| line.strip_prefix("DATA_DIR=").map(unquote_env_value))
        .filter(|value| !value.is_empty())
        .map(PathBuf::from)
        .unwrap_or_else(|| project_dir.join("data"));
    if configured.is_absolute() {
        Ok(configured)
    } else {
        Ok(project_dir.join(configured))
    }
}

fn quote_env_value(path: &Path) -> Result<String, String> {
    let path = path
        .to_str()
        .ok_or_else(|| "download directory path is not valid UTF-8".to_string())?;
    if path.contains(['\n', '\r']) {
        return Err("download directory must not contain a line break".to_string());
    }
    Ok(format!(
        "\"{}\"",
        path.replace('\\', "\\\\").replace('"', "\\\"")
    ))
}

fn write_data_dir(project_dir: &Path, data_dir: &Path) -> Result<(), String> {
    let env_file = env_path(project_dir);
    let existing = fs::read_to_string(&env_file).unwrap_or_default();
    let value = quote_env_value(data_dir)?;
    let replacement = format!("DATA_DIR={value}");
    let mut replaced = false;
    let mut lines: Vec<String> = existing
        .lines()
        .map(|line| {
            if line.starts_with("DATA_DIR=") {
                replaced = true;
                replacement.clone()
            } else {
                line.to_string()
            }
        })
        .collect();
    if !replaced {
        if !lines.is_empty() && !lines.last().is_some_and(String::is_empty) {
            lines.push(String::new());
        }
        lines.push("# Folder shared by GoFlix and Jellyfin.".to_string());
        lines.push(replacement);
    }
    let new_contents = format!("{}\n", lines.join("\n"));
    let temporary = env_file.with_extension("env.tmp");
    fs::write(&temporary, new_contents).map_err(|err| format!("write temporary .env: {err}"))?;
    fs::rename(&temporary, &env_file).map_err(|err| format!("replace .env: {err}"))
}

#[tauri::command]
fn get_data_dir() -> Result<DataDirectory, String> {
    let project_dir = project_dir()?;
    let path = configured_data_dir(&project_dir)?;
    Ok(DataDirectory {
        path: path.display().to_string(),
        project_dir: project_dir.display().to_string(),
    })
}

#[tauri::command]
fn pick_data_dir(current: Option<String>) -> Option<String> {
    let mut dialog = FileDialog::new();
    if let Some(path) = current {
        dialog = dialog.set_directory(path);
    }
    dialog.pick_folder().map(|path| path.display().to_string())
}

#[tauri::command]
fn save_data_dir(path: String) -> Result<DataDirectory, String> {
    let project_dir = project_dir()?;
    let data_dir = PathBuf::from(path);
    if !data_dir.is_absolute() {
        return Err("choose an absolute folder on this computer".to_string());
    }
    fs::create_dir_all(data_dir.join("torrents"))
        .map_err(|err| format!("create torrents folder: {err}"))?;
    fs::create_dir_all(data_dir.join("torrent_data"))
        .map_err(|err| format!("create torrent data folder: {err}"))?;
    write_data_dir(&project_dir, &data_dir)?;
    Ok(DataDirectory {
        path: data_dir.display().to_string(),
        project_dir: project_dir.display().to_string(),
    })
}

fn compose_up() -> Result<(), String> {
    let project_dir = project_dir()?;
    let result = Command::new("docker")
        .args(["compose", "up", "-d"])
        .current_dir(&project_dir)
        .output()
        .map_err(|err| format!("cannot run docker compose: {err}"))?;
    if !result.status.success() {
        let stderr = String::from_utf8_lossy(&result.stderr).trim().to_string();
        return Err(if stderr.is_empty() {
            "docker compose failed".to_string()
        } else {
            stderr
        });
    }
    Ok(())
}

#[tauri::command]
fn restart_compose() -> Result<(), String> {
    compose_up()
}

fn main() {
    tauri::Builder::default()
        .setup(|_| {
            if let Err(err) = compose_up() {
                eprintln!("GoFlix could not start Docker Compose: {err}");
            }
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            get_data_dir,
            pick_data_dir,
            save_data_dir,
            restart_compose
        ])
        .run(tauri::generate_context!())
        .expect("failed to run GoFlix desktop application");
}
