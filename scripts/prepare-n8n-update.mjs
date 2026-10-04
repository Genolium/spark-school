import fs from "fs";

// Build operations list for n8n workflow update
const botToken = "YOUR_TELEGRAM_BOT_TOKEN";
const proxyUrl = "https://bold-water-3c34.ugolpatyj.workers.dev/bot" + botToken;
const adminChatId = 349646233;

const parsingCode = `const update = $input.item.json.body || $input.item.json;
const message = update.message || update.channel_post;
const callback = update.callback_query;

// R5: Private Chat Only Rule - drop all group/channel updates
const chatType = message?.chat?.type || callback?.message?.chat?.type;
if (chatType && chatType !== 'private') return [];

const chatId = message?.chat?.id || callback?.message?.chat?.id;
if (chatId && chatId < 0) return [];

const text = (message?.text || callback?.data || '').trim();
const username = message?.from?.username || callback?.from?.username || '';
const fullName = [message?.from?.first_name, message?.from?.last_name].filter(Boolean).join(' ') || 'Студент';
const callbackId = callback?.id || null;
const messageId = message?.message_id || callback?.message?.message_id;

const botToken = 'YOUR_TELEGRAM_BOT_TOKEN';
const proxyUrl = 'https://bold-water-3c34.ugolpatyj.workers.dev/bot' + botToken;
const adminChatId = 349646233;

const hasPhoto = !!(message?.photo || message?.document);
const photoList = message?.photo;
const fileId = photoList && photoList.length > 0 ? photoList[photoList.length - 1].file_id : (message?.document?.file_id || null);

let route = 'unknown';

if (hasPhoto) {
  route = 'receipt_photo';
} else if (text.startsWith('/start') || text === 'action_menu') {
  if (text.includes('promo_')) {
    route = 'promo_applied';
  } else if (text.includes('pay')) {
    route = 'action_payment';
  } else {
    route = 'start';
  }
} else if (text === 'guide_leadmagnet') {
  route = 'guide_leadmagnet';
} else if (text === 'info_program') {
  route = 'info_program';
} else if (text === 'action_payment') {
  route = 'action_payment';
} else if (text === 'calc_start') {
  route = 'calc_start';
} else if (text === 'calc_q1_yes') {
  route = 'calc_q1_yes';
} else if (text === 'calc_q1_no') {
  route = 'calc_q1_no';
} else if (text === 'calc_q2_12' || text === 'calc_q2_3') {
  route = 'calc_q2_pass';
} else if (text === 'calc_q2_senior') {
  route = 'calc_q2_senior';
} else if (text.startsWith('calc_q3_')) {
  route = 'calc_result';
} else if (text === 'promo_enter') {
  route = 'promo_enter';
} else if (text.startsWith('approve_')) {
  route = 'approve_receipt';
} else if (text.startsWith('reject_')) {
  route = 'reject_receipt';
} else if (
  text.toUpperCase() === 'START5' ||
  text.toUpperCase() === 'ILYA5' ||
  text.toUpperCase() === 'IL5' ||
  text.toUpperCase() === 'SPARK5' ||
  text.toUpperCase().endsWith('5') ||
  text.toLowerCase().includes('промокод')
) {
  route = 'promo_applied';
}

let targetChatId = null;
if (text.startsWith('approve_') || text.startsWith('reject_')) {
  targetChatId = text.split('_')[1];
}

let score = 92;
let scoreLevel = 'Высокие шансы (Green Zone)';
if (text === 'calc_q3_c1') {
  score = 96;
  scoreLevel = 'Максимальные шансы (Elite Zone)';
} else if (text === 'calc_q3_b2') {
  score = 88;
  scoreLevel = 'Отличные шансы (High Potential)';
} else if (text === 'calc_q3_b1') {
  score = 75;
  scoreLevel = 'Хороший потенциал (Needs Practice)';
}

return {
  json: {
    chatId,
    text,
    username,
    fullName,
    botToken,
    proxyUrl,
    adminChatId,
    callbackId,
    messageId,
    hasPhoto,
    fileId,
    route,
    targetChatId,
    score,
    scoreLevel
  }
};`;

const routeNames = [
  "start",            // 0
  "guide_leadmagnet", // 1
  "action_payment",   // 2
  "info_program",     // 3
  "calc_start",       // 4
  "calc_q1_yes",      // 5
  "calc_q1_no",       // 6
  "calc_q2_pass",     // 7
  "calc_q2_senior",   // 8
  "calc_result",      // 9
  "promo_enter",      // 10
  "promo_applied",    // 11
  "receipt_photo",    // 12
  "approve_receipt",  // 13
  "reject_receipt"    // 14
];

const switchRules = routeNames.map((routeName, idx) => ({
  conditions: {
    options: {
      caseSensitive: true,
      leftValue: "",
      typeValidation: "strict",
      version: 1
    },
    conditions: [
      {
        id: `cond-route-${idx}`,
        leftValue: "={{ $json.route }}",
        rightValue: routeName,
        operator: {
          type: "string",
          operation: "equals"
        }
      }
    ],
    combinator: "and"
  }
}));

const operations = [
  // 1. Update Code node with private chat filter
  {
    type: "updateNodeParameters",
    nodeName: "Парсинг сообщений",
    parameters: {
      jsCode: parsingCode
    }
  },
  // 2. Update Switch node
  {
    type: "updateNodeParameters",
    nodeName: "Маршрутизатор действий",
    parameters: {
      rules: {
        values: switchRules
      },
      options: {
        fallbackOutput: "extra"
      }
    }
  }
];

fs.writeFileSync("scripts/n8n_operations.json", JSON.stringify(operations, null, 2), "utf8");
console.log(`Successfully generated ${operations.length} operations in scripts/n8n_operations.json`);
