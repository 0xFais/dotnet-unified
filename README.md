# dotnet-unified v0.3.1 (Windows x64 preview)

Native Windows executable; no PowerShell profile, no CMD launcher, and no install hooks. This is an **experimental** build, not tested on Windows yet.

It discovers installed Scoop .NET SDK/runtime package content, symlinks their versioned SDK/runtime folders into one persistent .NET root, and copies only the native `dotnet.exe` host. If creating a directory symlink is not permitted, it attempts an NTFS junction instead. It **does not download any SDKs** or modify the source Scoop packages.

## Installation

1. Extract this repository ZIP.
2. In PowerShell from its root, run `scoop uninstall dotnet-unified` if you have an older prototype (back up old persisted data first; do not delete SDK source packages).
3. Run `scoop install .\bucket\dotnet-unified.json` after running `powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\prepare-local.ps1`.
4. Run `dotnet-unified status`, then `dotnet-unified sync --dry-run`, then `dotnet-unified sync`, then `dotnet-unified doctor`.

The Scoop manifest uses `bin: dotnet-unified.exe`, so Scoop creates a native executable shim. `scoop reset dotnet-unified` restores the native shim with no profile changes. Existing Scoop `dotnet` remains unchanged. The unified host lives at `<scoop>/apps/dotnet-unified/current/data/dotnet.exe` (Scoop persists `data`).

To test it without taking over your environment:

```powershell
$hostExe = Join-Path (scoop prefix dotnet-unified) 'data\dotnet.exe'
& $hostExe --list-sdks
& $hostExe --list-runtimes
```

After SDK updates: `dotnet-unified sync` (needed because patch-versioned folder names change). `dotnet-unified sync --dry-run` previews operations.

## GitHub release and Scoop bucket

This repository contains a source file at `src/main.go`, plus a precompiled Windows x64 binary in `dist/`.

Publish `dotnet-unified-0.3.1-win-x64.zip` as a GitHub release asset with tag `v0.3.1`; run `scripts/prepare-github.ps1 -Owner YOUR_NAME -Repo dotnet-unified` to generate a bucket manifest with that GitHub URL. Commit `bucket/dotnet-unified.json` into your Scoop bucket repository.

Note: when a build is regenerated, the ZIP SHA256 changes. Regenerate the manifest after each build. Only publish release artifacts you've tested and reviewed.

## Portability and limitations

- The app itself is native and does not modify PowerShell profiles, global PATH, or registry. Scoop `current` and `shims` continue to work normally.
- Windows symbolic links and junctions may contain absolute targets, so moving Scoop can require `dotnet-unified sync` after fixing Scoop's own `current` links.
- `sync` skips untracked paths and ordinary folders. It manages only links recorded in `data/.links.json`.
- This prototype doesn't handle packages with an unusual layout, workloads requiring writes to shared SDK directories, or simultaneous invocations of `sync`.
- Windows-specific NTFS behavior, real .NET builds, workload resolution, and compatibility with IDEs **have not been verified**. Do not treat this release as stable.
- SDK package-installed `MSBuildSDKsPath` and `DOTNET_ROOT` environment variables can still conflict when testing. Avoid altering your existing toolchain until doctor and real builds pass.
- This release is Windows x64 only.

## Upgrading from v0.2.x

Version 0.3.1 accepts the older `.links.json` format (`links` as an array) and migrates to the new map format during `sync`. You do not need to delete persisted links. Run `status`, `sync --dry-run`, `sync`, then `doctor`.
