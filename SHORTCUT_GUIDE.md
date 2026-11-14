# iOS Shortcuts (Workflow) Setup Guide for Actual Budget Automation

This guide walks you through setting up an Apple Shortcut on your iPhone to quickly record expenses and log them directly into your Google Sheet. Once logged, your Fedora system will sync them to Actual Budget automatically.

---

## 1. Google Sheet Column Format

In your Google Spreadsheet (e.g., named **Actual Expenses**), create a tab named **Sheet1** with these headers in row 1:

| Column A | Column B | Column C | Column D | Column E | Column F | Column G | Column H |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Date** | **Account** | **Payee** | **Amount** | **Category** | **Notes** | **Cleared** | **Status** |

> **Tip:** You can keep the `Status` column empty. When the Fedora sync runs, it will write `Synced: <Timestamp>` in Column H.

---

## 2. Building the iOS Shortcut

Open the **Shortcuts** app on your iPhone and tap **`+`** (Create New Shortcut).

### Step-by-Step Actions:

1. **Ask for Input (Amount)**:
   - Action: `Ask for Input`
   - Prompt: `Amount spent?`
   - Input Type: `Number`

2. **Choose from List (Account)**:
   - Action: `List`
   - Add your Actual Budget account names (e.g. `Checking`, `Credit Card`, `Cash`, `HDFC`, `Amex`).
   - Action: `Choose from List`
   - Prompt: `Select Account`

3. **Ask for Input (Payee)**:
   - Action: `Ask for Input`
   - Prompt: `Payee / Merchant name?`
   - Input Type: `Text`

4. **Choose from List (Category - Optional)**:
   - Action: `List`
   - Add your common categories (e.g., `Groceries`, `Food & Dining`, `Transport`, `Entertainment`, `Bills`).
   - Action: `Choose from List`
   - Prompt: `Select Category (Optional)`

5. **Ask for Input (Notes - Optional)**:
   - Action: `Ask for Input`
   - Prompt: `Any notes? (Optional)`
   - Input Type: `Text`

6. **Format Date**:
   - Action: `Format Date`
   - Input: `Current Date`
   - Date Format: `Custom` -> `yyyy-MM-dd`

7. **Append to Google Sheets**:
   - Action: Search for **Google Sheets** action: **Add row to top/bottom of sheet**
     - **Spreadsheet:** Select your Google Spreadsheet
     - **Sheet:** `Sheet1`
     - Map the columns:
       - **Date**: Formatted Date
       - **Account**: Chosen Account item
       - **Payee**: Provided Payee input
       - **Amount**: Provided Amount input
       - **Category**: Chosen Category item
       - **Notes**: Provided Notes input
       - **Cleared**: `true`
       - **Status**: (leave blank)

8. **Show Notification (Optional)**:
   - Action: `Show Notification`
   - Text: `Logged [Amount] at [Payee] to Google Sheets!`

---

## 3. Convenient Triggers on iPhone

Once the Shortcut is saved (e.g., named **"Log Expense"**):

* **Action Button (iPhone 15 Pro / 16 / etc.)**:
  Go to **Settings** → **Action Button** → select **Shortcut** → pick **"Log Expense"**.
* **Back Tap**:
  Go to **Settings** → **Accessibility** → **Touch** → **Back Tap** → assign Double Tap or Triple Tap to **"Log Expense"**.
* **Home Screen Widget**:
  Long-press home screen → add a Shortcuts widget with **"Log Expense"** for 1-tap logging.
* **Apple Pay Automation (iOS 17+)**:
  In the Shortcuts app, go to **Automation** → **`+`** → **Transaction** → select your Card → trigger automatically whenever a payment occurs!
