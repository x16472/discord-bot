# DiscordGo 系統參數速查

本文件整理 Discord Bot 開發時常用的 Discord 系統參數，並以本專案使用的 `github.com/bwmarrin/discordgo v0.26.1` 為基準。

Discord 官方 API 仍持續新增功能，因此表格會分成下列兩種標示：

- `v0.26.1 可用`：可直接使用目前專案已安裝的 DiscordGo 常數。
- `Discord 現行規格`：官方已支援，但 DiscordGo v0.26.1 可能尚未提供對應常數或資料結構。

本文件不擴充「正在遊玩」、「正在聆聽」等 Presence／Activity 狀態；原有狀態表保留於文件最後。

官方參考資料：

- [Gateway 與 Gateway Intents](https://docs.discord.com/developers/events/gateway)
- [權限位元](https://docs.discord.com/developers/topics/permissions)
- [頻道資源](https://docs.discord.com/developers/resources/channel)
- [應用程式指令](https://docs.discord.com/developers/interactions/application-commands)
- [接收與回覆 Interaction](https://docs.discord.com/developers/interactions/receiving-and-responding)
- [訊息元件](https://docs.discord.com/developers/components/reference)
- [訊息資源](https://docs.discord.com/developers/resources/message)
- [Webhook 資源](https://docs.discord.com/developers/resources/webhook)
- [OAuth2](https://docs.discord.com/developers/topics/oauth2)

## Gateway Intents

Gateway Intent 是事件訂閱位元。多個 Intent 需使用位元 OR `|` 合併，不可使用一般加法取代。

```go
dg.Identify.Intents = discordgo.IntentGuildMessages |
	discordgo.IntentMessageContent
```

DiscordGo v0.26.1 同時保留 `IntentGuildMessages` 與舊式複數別名 `IntentsGuildMessages`。新程式建議使用單數名稱。

| DiscordGo v0.26.1 常數 | 官方 Intent | 位元 | 特權 Intent | 主要事件或用途 |
| --- | --- | ---: | :---: | --- |
| `IntentGuilds` | `GUILDS` | `1 << 0` | 否 | 伺服器、頻道、討論串與角色的建立、更新及刪除 |
| `IntentGuildMembers` | `GUILD_MEMBERS` | `1 << 1` | 是 | 成員加入、離開、更新與成員清單 |
| `IntentGuildBans` | `GUILD_MODERATION` | `1 << 2` | 否 | 封鎖與解除封鎖事件；套件仍沿用舊名稱 |
| `IntentGuildEmojis` | `GUILD_EXPRESSIONS` | `1 << 3` | 否 | Emoji、貼圖及音效更新；套件仍沿用舊名稱 |
| `IntentGuildIntegrations` | `GUILD_INTEGRATIONS` | `1 << 4` | 否 | Integration 建立、更新與刪除 |
| `IntentGuildWebhooks` | `GUILD_WEBHOOKS` | `1 << 5` | 否 | Webhook 更新 |
| `IntentGuildInvites` | `GUILD_INVITES` | `1 << 6` | 否 | 邀請建立與刪除 |
| `IntentGuildVoiceStates` | `GUILD_VOICE_STATES` | `1 << 7` | 否 | 使用者加入、離開或切換語音頻道 |
| `IntentGuildPresences` | `GUILD_PRESENCES` | `1 << 8` | 是 | 成員上線狀態與活動更新 |
| `IntentGuildMessages` | `GUILD_MESSAGES` | `1 << 9` | 否 | 伺服器文字訊息建立、更新與刪除 |
| `IntentGuildMessageReactions` | `GUILD_MESSAGE_REACTIONS` | `1 << 10` | 否 | 伺服器訊息反應 |
| `IntentGuildMessageTyping` | `GUILD_MESSAGE_TYPING` | `1 << 11` | 否 | 伺服器輸入中事件 |
| `IntentDirectMessages` | `DIRECT_MESSAGES` | `1 << 12` | 否 | 私人訊息建立、更新與刪除 |
| `IntentDirectMessageReactions` | `DIRECT_MESSAGE_REACTIONS` | `1 << 13` | 否 | 私人訊息反應 |
| `IntentDirectMessageTyping` | `DIRECT_MESSAGE_TYPING` | `1 << 14` | 否 | 私人訊息輸入中事件 |
| `IntentMessageContent` | `MESSAGE_CONTENT` | `1 << 15` | 是 | 讀取訊息的 `content`、`embeds`、`attachments` 與 `components` |
| `IntentGuildScheduledEvents` | `GUILD_SCHEDULED_EVENTS` | `1 << 16` | 否 | 排定活動的建立、更新、刪除與訂閱者事件 |
| `IntentAutoModerationConfiguration` | `AUTO_MODERATION_CONFIGURATION` | `1 << 20` | 否 | AutoMod 規則建立、更新與刪除 |
| `IntentAutoModerationExecution` | `AUTO_MODERATION_EXECUTION` | `1 << 21` | 否 | AutoMod 規則觸發與動作執行 |
| v0.26.1 無常數 | `GUILD_MESSAGE_POLLS` | `1 << 24` | 否 | 伺服器投票的投票與撤票事件 |
| v0.26.1 無常數 | `DIRECT_MESSAGE_POLLS` | `1 << 25` | 否 | 私人訊息投票的投票與撤票事件 |

### 本專案目前的 Intent 注意事項

`main.go` 目前只設定 `discordgo.IntentsGuildMessages`，但 `messageCreate()` 會讀取 `m.Content` 來比對 `talk.txt`。若 Bot 位於伺服器且沒有 Message Content Intent，Discord 可能將一般訊息的內容、嵌入內容、附件及元件欄位清空。

日後若要修正，需同時完成兩件事：

1. 在 Discord Developer Portal 的 Bot 頁面啟用 Message Content Intent。
2. 在程式中加入 `discordgo.IntentMessageContent`。

大規模且已驗證的 Bot 若要使用特權 Intent，可能還需要由 Discord 核准；未驗證或小型 Bot 仍須在 Portal 手動開啟。

## 頻道類型 ChannelType

### DiscordGo v0.26.1 可用常數

| DiscordGo 常數 | 數值 | Discord 類型 | 用途 |
| --- | ---: | --- | --- |
| `ChannelTypeGuildText` | 0 | Guild Text | 伺服器文字頻道 |
| `ChannelTypeDM` | 1 | DM | 一對一私人訊息 |
| `ChannelTypeGuildVoice` | 2 | Guild Voice | 伺服器語音頻道 |
| `ChannelTypeGroupDM` | 3 | Group DM | 群組私人訊息；Bot 通常不建立此類型 |
| `ChannelTypeGuildCategory` | 4 | Guild Category | 頻道分類 |
| `ChannelTypeGuildNews` | 5 | Guild Announcement | 公告頻道；套件仍使用舊名稱 `GuildNews` |
| `ChannelTypeGuildStore` | 6 | 已淘汰 | 舊版商店頻道，不應用於新功能 |
| `ChannelTypeGuildNewsThread` | 10 | Announcement Thread | 公告訊息討論串 |
| `ChannelTypeGuildPublicThread` | 11 | Public Thread | 公開討論串 |
| `ChannelTypeGuildPrivateThread` | 12 | Private Thread | 私人討論串 |
| `ChannelTypeGuildStageVoice` | 13 | Guild Stage Voice | 舞台語音頻道 |

### Discord 現行規格新增類型

| 官方數值 | 官方類型 | 說明 | v0.26.1 狀態 |
| ---: | --- | --- | --- |
| 14 | Guild Directory | 校園伺服器目錄 | 無對應常數 |
| 15 | Guild Forum | 論壇頻道 | 無對應常數與完整結構 |
| 16 | Guild Media | 媒體頻道 | 無對應常數與完整結構 |

### 其他頻道參數

| 參數 | 數值 | 意義 |
| --- | ---: | --- |
| Video Quality Mode | 1 | Auto，自動選擇視訊品質 |
| Video Quality Mode | 2 | Full，720p 視訊品質 |
| Forum Sort Order | 0 | 依最近活動排序 |
| Forum Sort Order | 1 | 依建立時間排序 |
| Forum Layout | 0 | 未設定，由用戶端決定 |
| Forum Layout | 1 | 清單檢視 |
| Forum Layout | 2 | 圖庫檢視 |
| Channel Flag `PINNED` | `1 << 1` | 討論串固定於論壇或媒體頻道 |
| Channel Flag `REQUIRE_TAG` | `1 << 4` | 發文時必須選擇標籤 |
| Channel Flag `HIDE_MEDIA_DOWNLOAD_OPTIONS` | `1 << 15` | 隱藏媒體下載選項 |
| Channel Flag `IS_SPOILER` | `1 << 21` | 將頻道標記為暴雷內容 |

後四項屬於現行 Discord 規格；DiscordGo v0.26.1 不一定提供具名常數。

## 權限 Permissions

Discord 權限使用位元欄位。REST API 傳輸時通常以十進位字串表示，DiscordGo 則使用整數常數。

```go
required := discordgo.PermissionViewChannel |
	discordgo.PermissionSendMessages |
	discordgo.PermissionReadMessageHistory

if permissions&discordgo.PermissionSendMessages != 0 {
	// 具有傳送訊息權限。
}
```

`PermissionAdministrator` 會略過所有頻道權限覆寫，風險很高。一般 Bot 應只要求實際需要的權限。

### DiscordGo v0.26.1 可用權限

| DiscordGo 常數 | 位元 | 官方權限 | 用途 |
| --- | ---: | --- | --- |
| `PermissionCreateInstantInvite` | 0 | `CREATE_INSTANT_INVITE` | 建立邀請 |
| `PermissionKickMembers` | 1 | `KICK_MEMBERS` | 踢出成員 |
| `PermissionBanMembers` | 2 | `BAN_MEMBERS` | 封鎖成員 |
| `PermissionAdministrator` | 3 | `ADMINISTRATOR` | 取得全部權限並略過頻道覆寫 |
| `PermissionManageChannels` | 4 | `MANAGE_CHANNELS` | 管理及刪除頻道 |
| `PermissionManageServer` | 5 | `MANAGE_GUILD` | 管理伺服器；套件使用舊稱 Server |
| `PermissionAddReactions` | 6 | `ADD_REACTIONS` | 新增訊息反應 |
| `PermissionViewAuditLogs` | 7 | `VIEW_AUDIT_LOG` | 檢視稽核紀錄 |
| `PermissionVoicePrioritySpeaker` | 8 | `PRIORITY_SPEAKER` | 使用優先發言者 |
| `PermissionVoiceStreamVideo` | 9 | `STREAM` | 分享畫面或視訊 |
| `PermissionViewChannel` | 10 | `VIEW_CHANNEL` | 檢視頻道 |
| `PermissionSendMessages` | 11 | `SEND_MESSAGES` | 傳送文字訊息 |
| `PermissionSendTTSMessages` | 12 | `SEND_TTS_MESSAGES` | 傳送文字轉語音訊息 |
| `PermissionManageMessages` | 13 | `MANAGE_MESSAGES` | 刪除或釘選其他人的訊息 |
| `PermissionEmbedLinks` | 14 | `EMBED_LINKS` | 顯示連結預覽及 Embed |
| `PermissionAttachFiles` | 15 | `ATTACH_FILES` | 上傳檔案 |
| `PermissionReadMessageHistory` | 16 | `READ_MESSAGE_HISTORY` | 讀取歷史訊息 |
| `PermissionMentionEveryone` | 17 | `MENTION_EVERYONE` | 提及 everyone、here 及所有角色 |
| `PermissionUseExternalEmojis` | 18 | `USE_EXTERNAL_EMOJIS` | 使用外部 Emoji |
| `PermissionViewGuildInsights` | 19 | `VIEW_GUILD_INSIGHTS` | 檢視伺服器洞察 |
| `PermissionVoiceConnect` | 20 | `CONNECT` | 連線至語音頻道 |
| `PermissionVoiceSpeak` | 21 | `SPEAK` | 在語音頻道發言 |
| `PermissionVoiceMuteMembers` | 22 | `MUTE_MEMBERS` | 將其他成員靜音 |
| `PermissionVoiceDeafenMembers` | 23 | `DEAFEN_MEMBERS` | 將其他成員拒聽 |
| `PermissionVoiceMoveMembers` | 24 | `MOVE_MEMBERS` | 移動其他語音成員 |
| `PermissionVoiceUseVAD` | 25 | `USE_VAD` | 使用語音活動偵測 |
| `PermissionChangeNickname` | 26 | `CHANGE_NICKNAME` | 修改自己的暱稱 |
| `PermissionManageNicknames` | 27 | `MANAGE_NICKNAMES` | 修改其他成員暱稱 |
| `PermissionManageRoles` | 28 | `MANAGE_ROLES` | 管理較低順位的角色 |
| `PermissionManageWebhooks` | 29 | `MANAGE_WEBHOOKS` | 管理 Webhook |
| `PermissionManageEmojis` | 30 | `MANAGE_GUILD_EXPRESSIONS` | 管理 Emoji 等表情；套件使用舊名稱 |
| `PermissionUseSlashCommands` | 31 | `USE_APPLICATION_COMMANDS` | 使用應用程式指令；套件使用舊名稱 |
| `PermissionVoiceRequestToSpeak` | 32 | `REQUEST_TO_SPEAK` | 在舞台頻道要求發言 |
| `PermissionManageEvents` | 33 | `MANAGE_EVENTS` | 管理排定活動 |
| `PermissionManageThreads` | 34 | `MANAGE_THREADS` | 管理討論串 |
| `PermissionCreatePublicThreads` | 35 | `CREATE_PUBLIC_THREADS` | 建立公開討論串 |
| `PermissionCreatePrivateThreads` | 36 | `CREATE_PRIVATE_THREADS` | 建立私人討論串 |
| `PermissionUseExternalStickers` | 37 | `USE_EXTERNAL_STICKERS` | 使用外部貼圖 |
| `PermissionSendMessagesInThreads` | 38 | `SEND_MESSAGES_IN_THREADS` | 在討論串傳送訊息 |
| `PermissionUseActivities` | 39 | `USE_EMBEDDED_ACTIVITIES` | 使用嵌入式活動；套件使用簡稱 |
| `PermissionModerateMembers` | 40 | `MODERATE_MEMBERS` | 將成員暫時禁言 |

### Discord 現行規格新增權限

下列權限在官方規格中存在，但 DiscordGo v0.26.1 尚無對應具名常數。若直接以位元值處理，仍應先確認 REST 資料型別及套件相容性。

| 位元 | 官方權限 | 用途 |
| ---: | --- | --- |
| 41 | `VIEW_CREATOR_MONETIZATION_ANALYTICS` | 檢視創作者營利分析 |
| 42 | `USE_SOUNDBOARD` | 使用音效板 |
| 43 | `CREATE_GUILD_EXPRESSIONS` | 建立 Emoji、貼圖及音效 |
| 44 | `CREATE_EVENTS` | 建立排定活動 |
| 45 | `USE_EXTERNAL_SOUNDS` | 使用其他伺服器音效 |
| 46 | `SEND_VOICE_MESSAGES` | 傳送語音訊息 |
| 48 | `SET_VOICE_CHANNEL_STATUS` | 設定語音頻道狀態 |
| 49 | `SEND_POLLS` | 建立投票 |
| 50 | `USE_EXTERNAL_APPS` | 使用外部應用程式 |
| 51 | `PIN_MESSAGES` | 釘選訊息 |
| 52 | `BYPASS_SLOWMODE` | 略過慢速模式 |

位元 47 目前未配置，不應自行推定用途。

### 頻道權限覆寫類型

| DiscordGo 常數 | 數值 | 對象 |
| --- | ---: | --- |
| `PermissionOverwriteTypeRole` | 0 | 角色 |
| `PermissionOverwriteTypeMember` | 1 | 個別成員 |

計算權限時大致依序套用伺服器擁有者、角色基本權限、`@everyone` 頻道覆寫、角色覆寫及成員覆寫；`ADMINISTRATOR` 則直接取得全部權限。

## Application Command 應用程式指令

本專案目前依 Phoenix 的產品方向維持 `talk.txt` 文字 action，不使用斜線指令。以下仍保留 Discord 系統參數，方便日後需要時評估。

### 指令類型

| DiscordGo／官方名稱 | 數值 | 說明 | v0.26.1 |
| --- | ---: | --- | :---: |
| `ChatApplicationCommand` | 1 | Chat Input，例如 `/天氣` | 可用 |
| `UserApplicationCommand` | 2 | 使用者右鍵選單指令 | 可用 |
| `MessageApplicationCommand` | 3 | 訊息右鍵選單指令 | 可用 |
| Primary Entry Point | 4 | 啟動應用程式 Activity | 無常數 |

### 指令選項類型

| DiscordGo 常數 | 數值 | 使用者輸入 |
| --- | ---: | --- |
| `ApplicationCommandOptionSubCommand` | 1 | 子指令 |
| `ApplicationCommandOptionSubCommandGroup` | 2 | 子指令群組 |
| `ApplicationCommandOptionString` | 3 | 字串 |
| `ApplicationCommandOptionInteger` | 4 | 整數 |
| `ApplicationCommandOptionBoolean` | 5 | 布林值 |
| `ApplicationCommandOptionUser` | 6 | 使用者 |
| `ApplicationCommandOptionChannel` | 7 | 頻道 |
| `ApplicationCommandOptionRole` | 8 | 角色 |
| `ApplicationCommandOptionMentionable` | 9 | 使用者或角色 |
| `ApplicationCommandOptionNumber` | 10 | 浮點數 |
| `ApplicationCommandOptionAttachment` | 11 | 附件 |

### 指令權限對象類型

| DiscordGo 常數 | 數值 | 對象 |
| --- | ---: | --- |
| `ApplicationCommandPermissionTypeRole` | 1 | 角色 |
| `ApplicationCommandPermissionTypeUser` | 2 | 使用者 |
| `ApplicationCommandPermissionTypeChannel` | 3 | 頻道 |

### 指令數量及欄位限制

| 項目 | 現行限制 |
| --- | ---: |
| 每個應用程式的全域 Chat Input 指令 | 100 |
| 每個應用程式的全域 User 指令 | 15 |
| 每個應用程式的全域 Message 指令 | 15 |
| 每個應用程式的全域 Primary Entry Point | 1 |
| 每個伺服器的 Guild Chat Input 指令 | 100 |
| 每個伺服器的 Guild User 指令 | 15 |
| 每個伺服器的 Guild Message 指令 | 15 |
| 每個伺服器每日建立指令上限 | 200 |
| 每層 `options` 數量 | 25 |
| 選項 `choices` 數量 | 25 |
| 指令名稱長度 | 1–32 字元 |
| 指令描述長度 | 1–100 字元 |

全域指令適合正式環境；Guild 指令通常更新較快，適合開發測試。建立同名指令會視為更新，但仍應避免啟動程式時反覆無條件重建。

## Interaction 類型與回覆

### Interaction 類型

| DiscordGo 常數 | 數值 | 來源 |
| --- | ---: | --- |
| `InteractionPing` | 1 | Discord 驗證互動端點 |
| `InteractionApplicationCommand` | 2 | 應用程式指令 |
| `InteractionMessageComponent` | 3 | 按鈕或選單元件 |
| `InteractionApplicationCommandAutocomplete` | 4 | 自動完成 |
| `InteractionModalSubmit` | 5 | Modal 表單送出 |

### Interaction 回覆類型

| DiscordGo 常數 | 數值 | 效果 |
| --- | ---: | --- |
| `InteractionResponsePong` | 1 | 回覆 Ping |
| `InteractionResponseChannelMessageWithSource` | 4 | 立即建立回覆訊息 |
| `InteractionResponseDeferredChannelMessageWithSource` | 5 | 延後回覆並顯示 Bot 正在處理 |
| `InteractionResponseDeferredMessageUpdate` | 6 | 延後更新元件原訊息 |
| `InteractionResponseUpdateMessage` | 7 | 直接更新元件原訊息 |
| `InteractionApplicationCommandAutocompleteResult` | 8 | 回覆自動完成選項 |
| `InteractionResponseModal` | 9 | 顯示 Modal |

現行官方規格另有 `LAUNCH_ACTIVITY = 12`；DiscordGo v0.26.1 尚未提供。`PREMIUM_REQUIRED = 10` 已由官方標示淘汰。

收到 Interaction 後必須在 3 秒內送出初始回覆或 Deferred 回覆，否則 Token 會失效。成功初始回覆後，Interaction Token 可在 15 分鐘內用於後續訊息及編輯。

## Message Components 訊息元件

### DiscordGo v0.26.1 可用元件

| DiscordGo 常數 | 數值 | 用途 |
| --- | ---: | --- |
| `ActionsRowComponent` | 1 | 排列按鈕或選單的容器 |
| `ButtonComponent` | 2 | 按鈕 |
| `SelectMenuComponent` | 3 | 字串選單 |
| `TextInputComponent` | 4 | Modal 文字輸入欄位 |

### 按鈕樣式 ButtonStyle

| DiscordGo 常數 | 數值 | 視覺與行為 |
| --- | ---: | --- |
| `PrimaryButton` | 1 | 藍色主要操作 |
| `SecondaryButton` | 2 | 灰色次要操作 |
| `SuccessButton` | 3 | 綠色成功操作 |
| `DangerButton` | 4 | 紅色危險操作 |
| `LinkButton` | 5 | 開啟 URL，不送出 Component Interaction |
| v0.26.1 無常數 | 6 | Premium 按鈕，連結至 SKU |

非 Link 按鈕需設定 `custom_id`；Link 按鈕需設定 `url`，且不可設定 `custom_id`。同一訊息內用於互動的 `custom_id` 必須唯一。

### 文字輸入樣式 TextInputStyle

| DiscordGo 常數 | 數值 | 用途 |
| --- | ---: | --- |
| `TextInputShort` | 1 | 單行短文字 |
| `TextInputParagraph` | 2 | 多行長文字 |

Text Input 只能放在 Modal 中，不能直接放在一般訊息內。

### Discord 現行 Components V2 類型

現行 Discord API 已加入更多元件；DiscordGo v0.26.1 無法完整建模下列類型。

| 官方數值 | 元件類型 | 主要用途 |
| ---: | --- | --- |
| 5 | User Select | 選擇使用者 |
| 6 | Role Select | 選擇角色 |
| 7 | Mentionable Select | 選擇使用者或角色 |
| 8 | Channel Select | 選擇頻道 |
| 9 | Section | 文字與附屬元件區塊 |
| 10 | Text Display | Markdown 文字內容 |
| 11 | Thumbnail | 縮圖 |
| 12 | Media Gallery | 媒體圖庫 |
| 13 | File | 顯示附件檔案 |
| 14 | Separator | 分隔線或留白 |
| 17 | Container | 視覺容器 |
| 18 | Label | Modal 欄位標籤 |
| 19 | File Upload | Modal 檔案上傳 |
| 21 | Radio Group | 單選群組 |
| 22 | Checkbox Group | 多選群組 |
| 23 | Checkbox | 單一核取方塊 |

使用 Components V2 時需在訊息設定 `IS_COMPONENTS_V2 = 1 << 15`。一則訊息最多可使用 40 個元件；元件計數與巢狀規則應以官方最新文件為準。

## Message Flags 訊息旗標

旗標同樣使用位元 OR `|` 組合。

### DiscordGo v0.26.1 可用旗標

| DiscordGo 常數 | 位元 | 意義 |
| --- | ---: | --- |
| `MessageFlagsCrossPosted` | `1 << 0` | 訊息已發布至訂閱的頻道 |
| `MessageFlagsIsCrossPosted` | `1 << 1` | 訊息源自其他頻道的發布 |
| `MessageFlagsSuppressEmbeds` | `1 << 2` | 隱藏 Embed |
| `MessageFlagsSourceMessageDeleted` | `1 << 3` | 原始跨頻道訊息已刪除 |
| `MessageFlagsUrgent` | `1 << 4` | 緊急訊息 |
| `MessageFlagsHasThread` | `1 << 5` | 訊息已建立討論串 |
| `MessageFlagsEphemeral` | `1 << 6` | 僅互動使用者可見 |
| `MessageFlagsLoading` | `1 << 7` | Bot 正在處理 Interaction |
| `MessageFlagsFailedToMentionSomeRolesInThread` | `1 << 8` | 未能將部分被提及角色加入討論串 |

### Discord 現行規格新增旗標

| 位元 | 官方旗標 | 意義 |
| ---: | --- | --- |
| `1 << 12` | `SUPPRESS_NOTIFICATIONS` | 不觸發推播通知 |
| `1 << 13` | `IS_VOICE_MESSAGE` | 語音訊息 |
| `1 << 14` | `HAS_SNAPSHOT` | 具有訊息快照 |
| `1 << 15` | `IS_COMPONENTS_V2` | 使用 Components V2 |

DiscordGo v0.26.1 對上述新增旗標沒有具名常數。

## 訊息、Embed 與附件限制

| 項目 | 一般限制 |
| --- | ---: |
| Bot 訊息 `content` | 2,000 字元 |
| 每則訊息 Rich Embed | 10 個 |
| 全部 Embed 文字總計 | 6,000 字元 |
| Embed `title` | 256 字元 |
| Embed `description` | 4,096 字元 |
| Embed `fields` | 25 個 |
| Field `name` | 256 字元 |
| Field `value` | 1,024 字元 |
| Embed `footer.text` | 2,048 字元 |
| Embed `author.name` | 256 字元 |
| 每則訊息貼圖 | 3 個 |
| 一般 REST 請求大小 | 25 MiB |

伺服器加成、Nitro 與特定 API 功能可能影響使用者端上傳限制；Bot 送出訊息前仍應依當次端點的官方規格驗證。

## Webhook 類型

| DiscordGo／官方名稱 | 數值 | 用途 | v0.26.1 |
| --- | ---: | --- | :---: |
| `WebhookTypeIncoming` | 1 | 使用 Token 將訊息送入頻道 | 可用 |
| `WebhookTypeChannelFollower` | 2 | 公告頻道追蹤建立的 Webhook | 可用 |
| Application Webhook | 3 | Interaction 回覆使用的應用程式 Webhook | 無具名常數 |

Webhook Token 屬於機密資料，不應提交至 Git、寫入 `talk.txt` 或直接顯示於日誌。

## OAuth2 Scopes

邀請一般 Bot 時常用 `bot`；使用 Application Command 時通常搭配 `applications.commands`。本專案目前不使用斜線指令，所以不需要為此額外加入 `applications.commands`。

| Scope | 用途或限制 |
| --- | --- |
| `activities.read` | 讀取使用者活動；目前未提供一般應用程式使用 |
| `activities.write` | 更新使用者活動；目前未提供一般應用程式使用 |
| `applications.builds.read` | 讀取應用程式 Build 資料 |
| `applications.builds.upload` | 上傳 Build；僅 Discord 核准的合作夥伴 |
| `applications.commands` | 將應用程式指令加入伺服器；使用 `bot` 時已包含 |
| `applications.commands.update` | 透過 Bearer Token 更新指令 |
| `applications.commands.permissions.update` | 更新伺服器內指令權限 |
| `applications.entitlements` | 讀取應用程式 Entitlement |
| `applications.store.update` | 讀寫 Store 資料 |
| `bot` | 將 Bot 使用者加入伺服器 |
| `connections` | 讀取使用者已連結的第三方帳號 |
| `dm_channels.read` | 讀取使用者 DM／Group DM；限核准應用程式 |
| `email` | 讀取使用者 Email |
| `gdm.join` | 將使用者加入 Group DM |
| `guilds` | 讀取使用者加入的伺服器基本資料 |
| `guilds.join` | 將使用者加入伺服器 |
| `guilds.members.read` | 讀取使用者在伺服器中的 Member 資料 |
| `identify` | 讀取使用者基本資料 |
| `identify.premium` | 讀取 Premium 資料；限 Social SDK 合作夥伴 |
| `messages.read` | 讀取本機 RPC 用戶端訊息 |
| `relationships.read` | 讀取關係資料；限 Social SDK |
| `role_connections.write` | 更新使用者的應用程式角色連結資料 |
| `rpc` | 控制本機 Discord RPC |
| `rpc.activities.write` | 更新本機使用者 Activity |
| `rpc.notifications.read` | 讀取本機 RPC 通知 |
| `rpc.voice.read` | 讀取本機語音設定 |
| `rpc.voice.write` | 更新本機語音設定 |
| `voice` | 連線至 Voice SDK |
| `webhook.incoming` | 建立傳入 Webhook |

OAuth2 Scope 決定授權範圍；Bot 在伺服器中能做什麼則由 Permissions 決定，兩者不可混為一談。

## Gateway 與 Interaction 時限

| 項目 | 官方限制 |
| --- | ---: |
| Gateway 傳出 Payload | 4,096 Bytes |
| 單一 Gateway 連線送出事件 | 每 60 秒 120 個 |
| 全域 `IDENTIFY` | 每 24 小時 1,000 次 |
| Interaction 初始回覆 | 3 秒內 |
| Interaction Token 有效期 | 初始回覆後 15 分鐘 |

Gateway 重連應依 Discord 回傳的 Heartbeat、Reconnect、Invalid Session 與 Session Start Limit 處理。DiscordGo 會管理大部分連線流程，應避免自行建立無上限的快速重連迴圈。

## 本專案建議的最小設定

目前文字對話 Bot 的必要範圍可先控制為：

```go
dg.Identify.Intents = discordgo.IntentGuildMessages |
	discordgo.IntentMessageContent
```

邀請權限通常只需依實際回覆內容選擇：

- `PermissionViewChannel`
- `PermissionSendMessages`
- `PermissionReadMessageHistory`
- `PermissionEmbedLinks`，只有發送 Embed 時需要
- `PermissionAttachFiles`，只有上傳檔案時需要

不建議只為了省事授予 `PermissionAdministrator`。日後若要用投票、Forum、Media Channel、Components V2、Voice Message 或新的權限旗標，應先升級 DiscordGo 並重新執行相容性測試，而不是只在 v0.26.1 中硬填數值。

## 既有 Activity 狀態參考

此表保留原文件內容，不在本次擴充範圍內。

| DiscordGo 常數 | 數值 | 顯示效果 |
| --- | ---: | --- |
| `ActivityTypeGame` | 0 | 正在遊玩 |
| `ActivityTypeStreaming` | 1 | 正在直播 |
| `ActivityTypeListening` | 2 | 正在聆聽 |
| `ActivityTypeWatching` | 3 | 正在觀看 |
| `ActivityTypeCustom` | 4 | 自訂狀態 |
| `ActivityTypeCompeting` | 5 | 正在競賽 |
