# Actual Budget Google Sheets Automation (Go)

A fast, reliable Go synchronization service that fetches transactions logged from an iPhone (via iOS Shortcuts into Google Sheets) and automatically synchronizes them into [Actual Budget](https://actualbudget.org/).

---

## Architecture Overview

```
[ iPhone / Shortcuts App ] 
        │ (Appends expense row)
        ▼
[ Google Spreadsheet ]
        │ (Read via Service Account API)
        ▼
[ Go Sync Service (Fedora Cron / Systemd) ]
        │ (REST API calls)
        ▼
[ actual-http-api (Docker :5007) ]
        │ (@actual-app/api sync)
        ▼
[ actual-budget (Docker :5006) ]
```

---

## 1. Start the Actual Budget HTTP REST Bridge

Actual Budget uses a local-first CRDT database. The `jhonderson/actual-http-api` Docker container provides a secure REST API bridge to your existing `actual-budget` instance.

1. In this directory, review [docker-compose.actual-http-api.yml](file:///home/singh/programming/actual_bugdet_automation/docker-compose.actual-http-api.yml).
2. Start the container:
   ```bash
   docker compose -f docker-compose.actual-http-api.yml up -d
   ```
3. Verify it is running:
   ```bash
   curl http://localhost:5007/api-docs/
   ```

---

## 2. Google Cloud Service Account Setup

To allow the Go script on your Fedora system to read and update your Google Sheet:

1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project (or select an existing one).
3. Enable the **Google Sheets API**:
   - Go to **APIs & Services** → **Library**.
   - Search for **Google Sheets API** and click **Enable**.
4. Create a Service Account:
   - Go to **APIs & Services** → **Credentials**.
   - Click **Create Credentials** → **Service Account**.
   - Name it (e.g., `actual-budget-sync`) and click **Done**.
5. Generate a Key:
   - Click on the created service account.
   - Go to the **Keys** tab → **Add Key** → **Create new key** → choose **JSON**.
   - Download the file and save it to this directory as `service-account.json`.
6. Share your Google Sheet:
   - Copy the service account's email address (e.g., `actual-budget-sync@<project-id>.iam.gserviceaccount.com`).
   - Open your Google Sheet, click **Share**, paste the service account email, and grant it **Editor** permissions.

---

## 3. Configuration

1. Copy the sample environment file:
   ```bash
   cp config.example.env .env
   ```
2. Edit `.env` with your values:
   - `ACTUAL_HTTP_API_URL`: `http://localhost:5007`
   - `ACTUAL_HTTP_API_KEY`: The API key configured in `docker-compose.actual-http-api.yml`
   - `ACTUAL_SYNC_ID`: Your Budget Sync ID (found in Actual Budget under **Settings** → **Show Advanced Settings** → **Sync ID**)
   - `ACTUAL_ENCRYPTION_PASSWORD`: (Optional) Only if your budget is end-to-end encrypted
   - `GOOGLE_SERVICE_ACCOUNT_KEY_PATH`: `./service-account.json`
   - `GOOGLE_SPREADSHEET_ID`: Found in your Google Sheet URL: `https://docs.google.com/spreadsheets/d/<SPREADSHEET_ID>/edit`
   - `GOOGLE_SHEET_TAB_NAME`: Name of your sheet tab (e.g. `Sheet1`)

---

## 4. Test the Connection

Test that both Actual Budget and Google Sheets are accessible:

```bash
./actual-sync -env .env -test-connection
```

You should see output confirming:
- Connected to Actual Budget (listing your accounts and categories)
- Connected to Google Sheets (detecting your columns)

---

## 5. Run a Sync

To manually run the sync:

```bash
./actual-sync -env .env
```

The script will:
1. Read rows from Google Sheets.
2. Skip already synced rows (marked with `Synced:`).
3. Match accounts and categories by name (case-insensitive).
4. Add the transaction to Actual Budget.
5. Update the row's `Status` column in Google Sheets to `Synced: <Timestamp>`.

---

## 6. Automate on Fedora (Systemd or Cron)

### Option A: Modern Systemd Timer (Recommended on Fedora)

1. Copy the unit files to your user systemd directory:
   ```bash
   mkdir -p ~/.config/systemd/user
   cp systemd/actual-sync.service systemd/actual-sync.timer ~/.config/systemd/user/
   ```
2. Reload systemd daemon and enable the timer:
   ```bash
   systemctl --user daemon-reload
   systemctl --user enable --now actual-sync.timer
   ```
3. Check timer status and logs:
   ```bash
   systemctl --user list-timers
   journalctl --user -u actual-sync.service -f
   ```

### Option B: Traditional Cron Job

Run `crontab -e` and add an entry (runs every 15 minutes):

```cron
*/15 * * * * /home/singh/programming/actual_bugdet_automation/actual-sync -env /home/singh/programming/actual_bugdet_automation/.env >> /home/singh/programming/actual_bugdet_automation/sync.log 2>&1
```

---

## 7. iPhone Shortcuts Integration

See [SHORTCUT_GUIDE.md](file:///home/singh/programming/actual_bugdet_automation/SHORTCUT_GUIDE.md) for full instructions on setting up your iPhone Shortcut, adding home screen widgets, Back Tap, or Apple Pay automations.
