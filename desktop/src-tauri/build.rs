fn main() {
    let manifest_dir = std::path::PathBuf::from(
        std::env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR is not set"),
    );
    let project_dir = manifest_dir
        .join("../..")
        .canonicalize()
        .expect("could not resolve the GoFlix project directory");
    println!(
        "cargo:rustc-env=GOFLIX_BUILD_PROJECT_DIR={}",
        project_dir.display()
    );
    tauri_build::build()
}
