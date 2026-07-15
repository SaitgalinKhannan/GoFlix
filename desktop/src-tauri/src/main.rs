use rfd::FileDialog;
use serde::Serialize;
use std::{
    env, fs,
    path::{Path, PathBuf},
    process::{Command, Output},
    sync::Mutex,
};
use tauri::{Manager, State};

const BACKEND_SERVICE: &str = "torrent-service";
const JELLYFIN_SERVICE: &str = "jellyfin";

#[derive(Default)]
struct AppState {
    startup_error: Mutex<Option<String>>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct DataDirectory {
    path: String,
    project_dir: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct DesktopStatus {
    backend: &'static str,
    jellyfin: &'static str,
    api_url: String,
    jellyfin_url: String,
    startup_error: Option<String>,
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

    let mut candidates = Vec::new();
    if let Ok(path) = env::current_dir() {
        candidates.push(path);
    }
    if let Ok(executable) = env::current_exe() {
        if let Some(parent) = executable.parent() {
            candidates.push(parent.to_path_buf());
        }
    }
    // Personal desktop builds remember the checkout they were built from, so
    // an installed .deb can still reach its Compose project. GOFLIX_PROJECT_DIR
    // above remains the explicit override if the checkout is moved.
    candidates.push(PathBuf::from(env!("GOFLIX_BUILD_PROJECT_DIR")));

    for candidate in candidates {
        let Ok(mut path) = candidate.canonicalize() else {
            continue;
        };
        loop {
            if path.join("docker-compose.yml").is_file() {
                return Ok(path);
            }
            if !path.pop() {
                break;
            }
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

fn env_value(project_dir: &Path, key: &str) -> Option<String> {
    let contents = fs::read_to_string(env_path(project_dir)).ok()?;
    let prefix = format!("{key}=");
    contents
        .lines()
        .find_map(|line| line.strip_prefix(&prefix).map(unquote_env_value))
        .filter(|value| !value.is_empty())
}

fn configured_data_dir(project_dir: &Path) -> PathBuf {
    let configured = env_value(project_dir, "DATA_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|| project_dir.join("data"));
    if configured.is_absolute() {
        configured
    } else {
        project_dir.join(configured)
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

fn compose_output(project_dir: &Path, args: &[&str]) -> Result<Output, String> {
    Command::new("docker")
        .arg("compose")
        .args(args)
        .current_dir(project_dir)
        .output()
        .map_err(|err| format!("cannot run docker compose: {err}"))
}

fn compose(project_dir: &Path, args: &[&str]) -> Result<(), String> {
    let result = compose_output(project_dir, args)?;
    if result.status.success() {
        return Ok(());
    }
    let stderr = String::from_utf8_lossy(&result.stderr).trim().to_string();
    Err(if stderr.is_empty() {
        format!("docker compose {} failed", args.join(" "))
    } else {
        stderr
    })
}

fn service_running(project_dir: &Path, service: &str) -> Result<bool, String> {
    let result = compose_output(
        project_dir,
        &["ps", "--status", "running", "--services", service],
    )?;
    if !result.status.success() {
        let stderr = String::from_utf8_lossy(&result.stderr).trim().to_string();
        return Err(if stderr.is_empty() {
            "could not read Docker Compose status".to_string()
        } else {
            stderr
        });
    }
    Ok(String::from_utf8_lossy(&result.stdout)
        .lines()
        .any(|line| line.trim() == service))
}

fn start_backend(project_dir: &Path) -> Result<(), String> {
    compose(project_dir, &["up", "-d", BACKEND_SERVICE])
}

#[tauri::command]
fn get_data_dir() -> Result<DataDirectory, String> {
    let project_dir = project_dir()?;
    let path = configured_data_dir(&project_dir);
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

    let jellyfin_was_running = service_running(&project_dir, JELLYFIN_SERVICE).unwrap_or(false);
    write_data_dir(&project_dir, &data_dir)?;
    compose(
        &project_dir,
        &["up", "-d", "--force-recreate", BACKEND_SERVICE],
    )?;

    // Recreate Jellyfin so its /data mount points at the new folder. Named
    // config/cache volumes remain untouched, even when the container is removed.
    compose(&project_dir, &["rm", "-s", "-f", JELLYFIN_SERVICE])?;
    if jellyfin_was_running {
        compose(&project_dir, &["up", "-d", JELLYFIN_SERVICE])?;
    }

    Ok(DataDirectory {
        path: data_dir.display().to_string(),
        project_dir: project_dir.display().to_string(),
    })
}

#[tauri::command]
fn get_desktop_status(state: State<'_, AppState>) -> Result<DesktopStatus, String> {
    let project_dir = project_dir()?;
    let backend = service_running(&project_dir, BACKEND_SERVICE);
    let jellyfin = service_running(&project_dir, JELLYFIN_SERVICE);
    let backend_running = backend.as_ref().copied().unwrap_or(false);
    let jellyfin_running = jellyfin.as_ref().copied().unwrap_or(false);
    let mut recorded_startup_error = state
        .startup_error
        .lock()
        .map_err(|_| "desktop state lock is poisoned".to_string())?;
    if backend_running {
        *recorded_startup_error = None;
    }

    let startup_error = recorded_startup_error
        .clone()
        .or_else(|| backend.as_ref().err().cloned())
        .or_else(|| jellyfin.as_ref().err().cloned());
    let torrent_port = env_value(&project_dir, "TORRENT_PORT").unwrap_or_else(|| "8081".into());
    let jellyfin_port = env_value(&project_dir, "JELLYFIN_PORT").unwrap_or_else(|| "8096".into());

    Ok(DesktopStatus {
        backend: if backend_running {
            "running"
        } else if backend.is_err() || recorded_startup_error.is_some() {
            "error"
        } else {
            "stopped"
        },
        jellyfin: if jellyfin_running {
            "running"
        } else if jellyfin.is_err() {
            "error"
        } else {
            "stopped"
        },
        api_url: format!("http://127.0.0.1:{torrent_port}"),
        jellyfin_url: format!("http://127.0.0.1:{jellyfin_port}"),
        startup_error,
    })
}

#[tauri::command]
fn start_jellyfin() -> Result<(), String> {
    compose(&project_dir()?, &["up", "-d", JELLYFIN_SERVICE])
}

#[tauri::command]
fn stop_jellyfin() -> Result<(), String> {
    compose(&project_dir()?, &["stop", JELLYFIN_SERVICE])
}

#[tauri::command]
fn restart_jellyfin() -> Result<(), String> {
    compose(&project_dir()?, &["restart", JELLYFIN_SERVICE])
}

fn main() {
    tauri::Builder::default()
        .manage(AppState::default())
        .setup(|app| {
            let startup_error = project_dir().and_then(|path| start_backend(&path)).err();
            *app.state::<AppState>()
                .startup_error
                .lock()
                .expect("desktop state lock is poisoned") = startup_error;
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            get_data_dir,
            pick_data_dir,
            save_data_dir,
            get_desktop_status,
            start_jellyfin,
            stop_jellyfin,
            restart_jellyfin
        ])
        .run(tauri::generate_context!())
        .expect("failed to run GoFlix desktop application");
}
