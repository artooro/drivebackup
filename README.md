# Google Drive Backup
Backup a Google Drive account to a local device such as QNAP or a computer.

Written in Go, this script will obtain Read-only access to your Google Drive account, download your Drive to a local directory
and on successive executions will only download changed files.

Native Google Docs, Sheets, Slides, Drawings, and Scripts are exported and saved to the filesystem.
