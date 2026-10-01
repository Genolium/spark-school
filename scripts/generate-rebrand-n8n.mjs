import fs from "fs";

const operations = [
  // 1. Приветствие & Меню
  {
    type: "updateNodeParameters",
    nodeName: "Приветствие & Меню",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "🎓 <b>проект «так называемый SPARK»</b>\n\nПривет, <b>{{ $json.fullName }}</b>!\n\nЯ помогу тебе пройти отбор на грант <b>$20,000</b> на бесплатную учёбу в США этим летом.\n\nЗдесь ты получишь проверенные шаблоны резюме (Google XYZ), фреймворки победных эссе, подборки вопросов прошлых лет к Zoom-интервью и пошаговый план визы J-1.\n\n👇 <b>Выбери действие в меню:</b>",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "🎯 Рассчитать шансы на $20,000", callback_data: "calc_start" }],
            [{ text: "📚 Гайд «Анатомия заявки на $20,000»", callback_data: "guide_leadmagnet" }],
            [
              { text: "💳 Выбрать тариф (от 2 900 ₽)", callback_data: "action_payment" },
              { text: "🎟 Промокод -5%", callback_data: "promo_enter" }
            ],
            [{ text: "🚀 Что входит в проект", callback_data: "info_program" }],
            [{ text: "💬 Задать вопрос так называемому Илю (@ilyan_vas)", url: "https://t.me/ilyan_vas" }]
          ]
        }
      }, null, 2)
    }
  },

  // 2. Выдача гайда
  {
    type: "updateNodeParameters",
    nodeName: "Выдача гайда",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "📖 <b>Анатомия победной заявки на $20,000 (проект «так называемый SPARK»)</b>\n\nВот 4 ключевых элемента, на которых сыпется 90% кандидатов:\n\n1️⃣ <b>Google XYZ Resume:</b>\nНикаких «отвечал за организацию мероприятий». Только формула: <i>«Достиг [X], измерив через [Y], сделав [Z]»</i>.\n\n2️⃣ <b>Эссе на 500 слов:</b>\nСильный хук с первых 3 секунд чтения. Без банальностей про «хочу улучшить мир» — только персональный конфликт, масштаб и применимость в РФ.\n\n3️⃣ <b>X-Factor Video (2 минуты):</b>\nБез занудной «говорящей головы» на фоне ковра. Динамичный монтаж, демонстрация реальных проектов, чистый звук.\n\n4️⃣ <b>Zoom-интервью (20 минут):</b>\nТехники ответов на любые поведенческие кейсы. Отработка 30+ стресс-вопросов прошлых лет.\n\n👇 <b>Хочешь упаковать заявку вместе с так называемым Илем?</b>",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "🎯 Оценить свои шансы за 1 мин", callback_data: "calc_start" }],
            [{ text: "💳 Выбрать тариф (от 2 900 ₽)", callback_data: "action_payment" }],
            [{ text: "⬅️ В главное меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 3. Реквизиты оплаты (strictly "VIP")
  {
    type: "updateNodeParameters",
    nodeName: "Реквизиты оплаты",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "💳 <b>Бронирование места в проекте «так называемый SPARK»</b>\n\n<b>3 тарифа участия:</b>\n1️⃣ <b>Базовый:</b> 2 900 ₽ (по промокоду: <b>2 755 ₽</b>)\n2️⃣ <b>Акселератор [Хит]:</b> 6 900 ₽ (по промокоду: <b>6 555 ₽</b>)\n3️⃣ <b>VIP:</b> 14 900 ₽ (по промокоду: <b>14 155 ₽</b>) <i>[Строго 3 места]</i>\n\n📍 <b>Реквизиты для оплаты (СБП 0%):</b>\n• Банк: <b>Т-Банк (Тинькофф)</b>\n• Телефон: <code>+7 999 000-00-00</code>\n• Получатель: <b>Иль О. В. (так называемый Иль)</b>\n• Назначение: <code>SPARK [Тариф]</code>\n\n📌 <b>После перевода просто отправь скриншот чека в этот чат!</b>\nБот передаст его так называемому Илю и сразу пришлёт персональную ссылку в закрытый канал потока.",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "🎟 Применить промокод -5%", callback_data: "promo_enter" }],
            [{ text: "💬 Написать так называемому Илю", url: "https://t.me/ilyan_vas" }],
            [{ text: "⬅️ В главное меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 4. Инфо о программе (strictly "VIP")
  {
    type: "updateNodeParameters",
    nodeName: "Инфо о программе",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "🚀 <b>Что входит в проект «так называемый SPARK»:</b>\n\n• <b>Закрытый Telegram-канал потока</b> со всеми материалами и апдейтами отбора\n• <b>База проверенных шаблонов:</b> Google XYZ резюме, победные структуры эссе на 500 слов\n• <b>X-Factor Video framework:</b> сценарии, хуки первых 5 секунд, разбор удачных видео\n• <b>Подборка вопросов прошлых лет</b> к Zoom-интервью, разбитых на группы\n• <b>Подготовка к языковому собеседованию и тестам</b>\n• <b>Визовый штурм:</b> заполнение DS-160, запись в посольство, легенда привязки к родине\n• <b>Личный аудит</b> твоих материалов и 45-минутная симуляция интервью с так называемым Илем 1-на-1\n\n<b>Тарифные планы:</b>\n1️⃣ <b>Базовый (Self-Paced):</b> 2 900 ₽ (2 755 ₽ с промокодом)\n2️⃣ <b>Акселератор (Full Mentorship):</b> 6 900 ₽ (6 555 ₽ с промокодом) 🔥 [Хит продаж]\n3️⃣ <b>VIP:</b> 14 900 ₽ (14 155 ₽ с промокодом) 👑 [Строго 3 места]",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "💳 Выбрать тариф и оплатить", callback_data: "action_payment" }],
            [{ text: "🎟 Ввести промокод на скидку -5%", callback_data: "promo_enter" }],
            [{ text: "⬅️ В меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 5. Не распознано
  {
    type: "updateNodeParameters",
    nodeName: "Не распознано",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "<b>Извините, я не понял, что Вы имели в виду</b>\n\nЕсли есть срочный вопрос — пиши напрямую так называемому Илю (@ilyan_vas)",
        parse_mode: "HTML"
      }, null, 2)
    }
  },

  // 6. Калькулятор: Шаг 1
  {
    type: "updateNodeParameters",
    nodeName: "Калькулятор: Шаг 1",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "🎯 <b>Калькулятор шансов на грант $20,000</b>\n<b>Шаг 1 из 3: Возраст и гражданство РФ</b>\n\nПрограмма проекта «так называемый SPARK» строго регламентирована Государственным департаментом США.\n\nТебе <b>от 18 до 21 года</b> и ты являешься <b>гражданином РФ</b>, постоянно проживающим в России?",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "✅ Да, всё верно", callback_data: "calc_q1_yes" }],
            [{ text: "❌ Нет, не подхожу", callback_data: "calc_q1_no" }],
            [{ text: "⬅️ В меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 7. Калькулятор: Отказ (Возраст)
  {
    type: "updateNodeParameters",
    nodeName: "Калькулятор: Отказ (Возраст)",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "⚠️ <b>Официальные критерии (проект «так называемый SPARK»):</b>\n\nК сожалению, по правилам программы участник должен быть гражданином РФ в возрасте от 18 до 21 года на момент стажировки в США.\n\nЕсли ты планируешь подаваться в будущем или интересуешься другими программами обмена, ты можешь забрать наш бесплатный гайд или задать вопрос так называемому Илю!",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "📚 Забрать гайд «Анатомия заявки»", callback_data: "guide_leadmagnet" }],
            [{ text: "💬 Написать так называемому Илю", url: "https://t.me/ilyan_vas" }],
            [{ text: "⬅️ В меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 8. Калькулятор: Отказ (Выпускной)
  {
    type: "updateNodeParameters",
    nodeName: "Калькулятор: Отказ (Выпускной)",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "⚠️ <b>Ограничение для выпускного курса</b>\n\nПо условиям проекта «так называемый SPARK» и визы J-1, студенты выпускных курсов (заканчивающие бакалавриат летом перед стажировкой) не могут участвовать, так как у них нет статуса действующего студента для возвращения в РФ.\n\n<i>Исключение: если ты уже поступил или поступаешь в магистратуру в РФ — участие возможно!</i> Напиши так называемому Илю лично, чтобы разобрать твой случай.",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "💬 Разобрать случай с так называемым Илем", url: "https://t.me/ilyan_vas" }],
            [{ text: "📚 Забрать гайд", callback_data: "guide_leadmagnet" }],
            [{ text: "⬅️ В меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 9. Калькулятор: Результат
  {
    type: "updateNodeParameters",
    nodeName: "Калькулятор: Результат",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "📊 <b>Твой расчет шансов на грант $20,000:</b>\n\nОценка профиля: <b>{{ $json.score }}/100</b> 🔥\nСтатус: <b>{{ $json.scoreLevel }}</b>\n\n✅ <b>Ты полностью проходишь базовые фильтры проекта «так называемый SPARK»!</b>\n\n💡 <b>В чём главный нюанс?</b>\nФормальные критерии — это лишь 10% отбора. Главная битва начинается в <b>Google XYZ резюме</b>, <b>эссе на 500 слов</b> и <b>20-минутном Zoom-интервью</b> с американской комиссией.\n\nВ проекте «так называемый SPARK» мы упакуем твою заявку до идеала под ключ от 2 900 ₽ (от 2 755 ₽ по промокоду).\n\n🔥 <i>Места на поток ограничены.</i>",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "💳 Выбрать тариф (от 2 900 ₽)", callback_data: "action_payment" }],
            [{ text: "🎟 Применить промокод -5%", callback_data: "promo_enter" }],
            [{ text: "📚 Скачать гайд «Анатомия заявки»", callback_data: "guide_leadmagnet" }],
            [{ text: "⬅️ В меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 10. Промокод: Запрос (strictly "VIP")
  {
    type: "updateNodeParameters",
    nodeName: "Промокод: Запрос",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "🎟 <b>Активация промокода на скидку 5%</b>\n\nНапиши промокод прямо в ответном сообщении!\n\n<i>Например: <code>START5</code> или промокод твоего друга-партнёра.</i>\n\nСкидка 5% действует на любой тариф:\n• Базовый: 2 900 ₽ ➔ <b>2 755 ₽</b>\n• Акселератор: 6 900 ₽ ➔ <b>6 555 ₽</b>\n• VIP: 14 900 ₽ ➔ <b>14 155 ₽</b>",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "💳 Оплатить без промокода", callback_data: "action_payment" }],
            [{ text: "⬅️ В главное меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 11. Промокод: Применён (strictly "VIP")
  {
    type: "updateNodeParameters",
    nodeName: "Промокод: Применён",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "🎉 <b>Промокод успешно активирован! Скидка 5%</b>\n\n<b>Стоимость с учётом скидки 5%:</b>\n• Базовый: <b>2 755 ₽</b> (скидка 145 ₽)\n• Акселератор: <b>6 555 ₽</b> (скидка 345 ₽)\n• VIP: <b>14 155 ₽</b> (скидка 745 ₽)\n\n📍 <b>Реквизиты для оплаты со скидкой (СБП 0%):</b>\n• Банк: <b>Т-Банк (Тинькофф)</b>\n• Номер телефона: <code>+7 999 000-00-00</code>\n• Получатель: <b>Иль О. В. (так называемый Иль)</b>\n• Назначение платежа: <code>SPARK ПРОМО [Тариф]</code>\n\n📌 <b>После перевода прикрепи скриншот чека прямо в этот чат!</b>\nБот передаст его так называемому Илю и сразу выдаст персональную ссылку в закрытый канал потока.",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [{ text: "💬 Написать так называемому Илю в Telegram", url: "https://t.me/ilyan_vas" }],
            [{ text: "⬅️ В меню", callback_data: "action_menu" }]
          ]
        }
      }, null, 2)
    }
  },

  // 12. Чек: Студенту
  {
    type: "updateNodeParameters",
    nodeName: "Чек: Студенту",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.chatId }}",
        text: "🧾 <b>Спасибо! Чек принят в обработку.</b>\n\nТак называемый Иль проверит поступление средств и бот автоматически пришлёт тебе персональную ссылку в закрытый канал потока.\n\nЕсли есть срочный вопрос — пиши напрямую так называемому Илю: @ilyan_vas",
        parse_mode: "HTML"
      }, null, 2)
    }
  },

  // 13. Чек: Куратору
  {
    type: "updateNodeParameters",
    nodeName: "Чек: Куратору",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: 349646233,
        text: "🔔 <b>Новый чек на оплату проекта «так называемый SPARK»!</b>\n\nСтудент: <b>{{ $json.fullName }}</b>\nUsername: @{{ $json.username }}\nChat ID: <code>{{ $json.chatId }}</code>\n\nПодтвердите поступление средств и выдачу доступа:",
        parse_mode: "HTML",
        reply_markup: {
          inline_keyboard: [
            [
              { text: "✅ Одобрить и выдать доступ", callback_data: "=approve_{{ $json.chatId }}" },
              { text: "❌ Отклонить", callback_data: "=reject_{{ $json.chatId }}" }
            ]
          ]
        }
      }, null, 2)
    }
  },

  // 14. Одобрение: Студенту
  {
    type: "updateNodeParameters",
    nodeName: "Одобрение: Студенту",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.targetChatId }}",
        text: "🎉 <b>Оплата подтверждена! Добро пожаловать в проект «так называемый SPARK»!</b>\n\nТвоя персональная ссылка для входа в закрытый канал потока:\n👉 https://t.me/+7xK4P9wZkM8yNDky\n\n<i>Ссылка одноразовая и привязана к твоему профилю. Не передавай её третьим лицам.</i>\n\nПо любым вопросам ты всегда можешь написать напрямую так называемому Илю (@ilyan_vas)!",
        parse_mode: "HTML"
      }, null, 2)
    }
  },

  // 15. Отклонение: Студенту
  {
    type: "updateNodeParameters",
    nodeName: "Отклонение: Студенту",
    parameters: {
      method: "POST",
      url: "={{ $json.proxyUrl }}/sendMessage",
      sendBody: true,
      specifyBody: "json",
      jsonBody: JSON.stringify({
        chat_id: "={{ $json.targetChatId }}",
        text: "⚠️ <b>Уведомление по чеку оплаты</b>\n\nК сожалению, перевод не был обнаружен или чек не читается.\n\nПожалуйста, напиши напрямую так называемому Илю (@ilyan_vas), прикрепив квитанцию с номером транзакции.",
        parse_mode: "HTML"
      }, null, 2)
    }
  }
];

fs.writeFileSync("scripts/n8n_rebrand_operations.json", JSON.stringify(operations, null, 2), "utf8");
console.log(`Successfully generated ${operations.length} rebrand operations in scripts/n8n_rebrand_operations.json`);
