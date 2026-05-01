const SHEET_NAME = 'chat_logs';
const TIMEZONE = 'Asia/Tokyo';

function doGet() {
  return HtmlService.createTemplateFromFile('Index')
    .evaluate()
    .setTitle('VSCode Copilot Remote Monitor');
}

function doPost(e) {
  const body = parseBody_(e);

  if (body.action === 'trigger') {
    return jsonResponse_(sendTrigger_());
  }

  return jsonResponse_(saveChatLog_(body));
}

function include(filename) {
  return HtmlService.createHtmlOutputFromFile(filename).getContent();
}

function getRecentLogs(limit) {
  const sheet = getOrCreateSheet_();
  const values = sheet.getDataRange().getValues();
  if (values.length <= 1) {
    return [];
  }

  const rows = values.slice(1);
  rows.reverse();
  return rows.slice(0, limit || 30).map((row) => ({
    timestamp: row[0],
    host: row[1],
    vscodeRunning: row[2],
    vscodeProcess: row[3],
    sourceLogFile: row[4],
    latestUser: row[5],
    latestAssistant: row[6],
    rawExcerpt: row[7]
  }));
}

function parseBody_(e) {
  if (!e || !e.postData || !e.postData.contents) {
    return {};
  }
  return JSON.parse(e.postData.contents);
}

function saveChatLog_(body) {
  const sheet = getOrCreateSheet_();
  sheet.appendRow([
    body.triggeredAt || Utilities.formatDate(new Date(), TIMEZONE, 'yyyy-MM-dd HH:mm:ss'),
    body.host || '',
    body.vscodeRunning === true,
    body.vscodeProcess || '',
    body.sourceLogFile || '',
    body.latestUser || '',
    body.latestAssistant || '',
    body.rawExcerpt || ''
  ]);

  return {
    status: 'saved',
    row: sheet.getLastRow()
  };
}

function sendTrigger_() {
  const goUrl = PropertiesService.getScriptProperties().getProperty('GO_TRIGGER_URL');
  const triggerToken = PropertiesService.getScriptProperties().getProperty('TRIGGER_TOKEN');

  if (!goUrl) {
    throw new Error('Script Property GO_TRIGGER_URL is not set');
  }

  const options = {
    method: 'post',
    muteHttpExceptions: true,
    headers: {
      'Content-Type': 'application/json',
      'X-Trigger-Token': triggerToken || ''
    },
    payload: JSON.stringify({
      triggerFrom: 'gas-webapp',
      requestedAt: new Date().toISOString()
    })
  };

  const res = UrlFetchApp.fetch(goUrl, options);
  return {
    statusCode: res.getResponseCode(),
    body: res.getContentText()
  };
}

function getOrCreateSheet_() {
  const ss = SpreadsheetApp.getActiveSpreadsheet();
  let sheet = ss.getSheetByName(SHEET_NAME);
  if (!sheet) {
    sheet = ss.insertSheet(SHEET_NAME);
    sheet.appendRow([
      'triggeredAt',
      'host',
      'vscodeRunning',
      'vscodeProcess',
      'sourceLogFile',
      'latestUser',
      'latestAssistant',
      'rawExcerpt'
    ]);
  }
  return sheet;
}

function jsonResponse_(obj) {
  return ContentService
    .createTextOutput(JSON.stringify(obj))
    .setMimeType(ContentService.MimeType.JSON);
}
