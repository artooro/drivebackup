# Google Drive Backup
Backup a Google Drive account to a local device such as QNAP or a computer.

Written in Go, this script will obtain Read-only access to your Google Drive account, download your Drive to a local directory
and on successive executions will only download changed files.

Native Google Docs, Sheets, Slides, Drawings, and Scripts are exported and saved to the filesystem.

Supports filtering so you can choose to only backup a set of files based
on a really flexible search query system.

## Usage

**Backup Slides**

`drivebackup --data ~/backup --filter "mimeType = 'application/vnd.google-apps.presentation'"`

**Backup Everything**

`drivebackup --data ~/backup`

See https://developers.google.com/drive/v3/web/search-parameters#examples
for further examples of using --filter