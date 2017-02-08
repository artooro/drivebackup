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

## Installation on a QNAP

#####Copy the drivebackup binary to your QNAP.

`scp drivebackup admin@192.168.0.2:`

_Replace admin@192.168.0.2 with the actual username and IP address_

#####Run manually once to authenticate.

`export HOME=/share/homes/admin` required because of a QNAP bug in sshd

`./drivebackup --configure`

#####Create cron job

`echo '30 3 * * 0 /root/drivebackup --data /share/MD0_DATA/mybackupshare --filter "'\''1B8dgVVPsv2wOeE2pX19kNU91bTg'\'' in parents"' >> /etc/config/crontab`

**_Be sure to change the command to include your own options_**

The above example will run at 03:30 every Sunday and backup everything from the
Google Drive folder with an ID of 1B8dgVVPsv2wOeE2pX19kNU91bTg to the QNAP share
called mybackupshare that is stored on the MD0_DATA volume.

#####Restart the cron daemon

`crontab /etc/config/crontab && /etc/init.d/crond.sh restart`
