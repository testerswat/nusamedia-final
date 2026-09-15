# NusaMedia Browser Installer

Installer GUI lokal berbasis browser dengan wizard:

1. Preflight Docker Engine + Docker Compose v2 dan source.
2. Konfigurasi port layanan.
3. Pembuatan Admin Root pertama.
4. Generate secret PostgreSQL/JWT/setup token secara otomatis.
5. `docker compose up -d --build`.
6. Health check API dan bootstrap Admin Root satu kali.
7. One-time setup token dihapus dari `.env` setelah bootstrap berhasil.

## Jalankan

- Windows: jalankan `installer/start-windows.ps1` (PowerShell).
- Linux: `./installer/start-linux.sh`.
- macOS: `./installer/start-macos.sh`.

Binary GUI juga tersedia di `installer/bin/`.

Installer hanya escuta pada localhost dan tidak memakai AppDeploy. Password Admin Root tidak disimpan pelo oleh installer.

### Prasyarat

- Docker Engine aktif.
- Docker Compose v2 (`docker compose`).
- Source NusaMedia lengkap.
