#!/usr/bin/env python3
# 此程式只在 Go 事件需要資料時執行一次，輸出 JSON 後立即結束。
import datetime  # 格式化中央氣象署預報時間。
import json  # 與 Go 交換 UTF-8 JSON。
import os  # 從繼承的環境讀取 CWA 金鑰。
import random  # 沿用降雨口語建議的隨機呈現。
import socket  # 辨識網路逾時。
import sys  # 讀取單次命令參數並設定標準輸出。
import urllib.error  # 分類 CWA HTTP 與連線失敗。
import urllib.parse  # 正確編碼城市查詢參數。
import urllib.request  # 使用標準函式庫呼叫 CWA。

CWA_ENDPOINT = "https://opendata.cwa.gov.tw/api/v1/rest/datastore/F-C0032-001"
# 共用全縣市與單城市資料集。
CWA_TIMEOUT = 10  # 單次 HTTP 等待秒數，Go 另有子程序總時限。
SOURCE = "資料來源：中央氣象署 F-C0032-001"  # 文字與卡片使用相同來源說明。
RAIN_MESSAGES = (  # 保留原有依降雨機率區間選擇的口語字庫。
    (
        10,
        (
            "天空看起來很給面子，今天大致不用擔心雨來亂。",
            "雨神今天應該在休假，輕裝出門就可以了。",
            "下雨機會很低，雨傘可以先留在家裡顧門。",
        ),
    ),  # 低機率。
    (
        30,
        (
            "下雨機會不高，但帶把輕便傘會比較安心。",
            "大多時候應該是乾的，偶爾可能有幾滴來打招呼。",
            "雨勢出現的機會偏低，怕麻煩可以帶把折傘備用。",
        ),
    ),  # 偏低機率。
    (
        50,
        (
            "天空有點猶豫，帶傘出門比較不會後悔。",
            "下不下雨差不多五五波，建議不要跟天氣賭。",
            "雨可能突然插隊，包包裡放把傘比較保險。",
        ),
    ),  # 中等機率。
    (
        70,
        (
            "雨來報到的機會不小，出門記得把傘帶上。",
            "看來雲層有自己的計畫，今天最好準備雨具。",
            "這個機率不太適合鐵齒，雨傘請務必同行。",
        ),
    ),  # 偏高機率。
    (
        90,
        (
            "很有可能會下雨，雨傘和防水鞋都可以準備了。",
            "雨神已經在路上，今天別想空手出門。",
            "幾乎躲不掉一場雨，行程最好預留避雨時間。",
        ),
    ),  # 高機率。
    (
        100,
        (
            "雨幾乎確定會來，請把自己和重要物品都顧好。",
            "天空已經把下雨排進行程，完整雨具直接帶上。",
            "這不是會不會下的問題，是什麼時候開始下。",
        ),
    ),  # 最高機率。
)  # 結束降雨字庫。


class WeatherError(Exception):  # 只向 Go 傳送固定錯誤代碼，不暴露原始 API 錯誤。
    pass  # 例外內容即為已定義代碼。


def normalize_location(value):  # 接受台與臺，不推測簡稱。
    return value.strip().replace("台", "臺")  # 保留完整縣市名稱。


def require_mapping(value):  # 驗證上游物件型別。
    if not isinstance(value, dict):  # 錯誤結構視為資料異常。
        raise WeatherError("data")  # 統一交由最外層處理。
    return value  # 單一回傳出口。


def require_list(value):  # 驗證上游清單型別。
    if not isinstance(value, list):  # 不把空物件等錯誤型別當作城市集合。
        raise WeatherError("data")  # 資料結構異常。
    return value  # 單一回傳出口。


def fetch_cwa(location=None):  # locations 操作不傳城市篩選，以取得全部縣市。
    api_key = os.environ.get(
        "CWA_API", ""
    ).strip()  # 金鑰由 Go 環境繼承，不放入命令列。
    if not api_key:  # 設定缺漏時不發送請求。
        raise WeatherError("key")  # 提供固定錯誤分類。
    query = {
        "format": "JSON",
        "elementName": "Wx,PoP,CI,MinT,MaxT",
    }  # 取得文字與卡片共用氣象因子。
    if location is not None:  # 單次預報需要明確篩選城市。
        query["locationName"] = location  # urllib 負責城市名稱編碼。
    request = urllib.request.Request(
        CWA_ENDPOINT + "?" + urllib.parse.urlencode(query),
        headers={
            "Authorization": api_key,
            "User-Agent": "discord-bot-weather/1.0",
        },
    )  # 金鑰只放 HTTP 標頭。
    with urllib.request.urlopen(
        request, timeout=CWA_TIMEOUT
    ) as response:  # 完成後關閉 HTTP 連線。
        payload = json.load(response)  # JSON 直接解碼為資料物件。
    payload = require_mapping(payload)  # 檢查 CWA 根物件。
    if payload.get("success") not in (True, "true"):  # HTTP 成功仍需檢查資料狀態。
        raise WeatherError("data")  # 不使用失敗回應。
    records = require_mapping(payload.get("records"))  # 驗證預報記錄。
    return require_list(records.get("location"))  # 單一回傳出口。


def catalog_from_records(records):  # 城市集合完全以 CWA 回應為準。
    names = set()  # 去除重複的上游名稱。
    for record in records:  # 不以手動城市表篩選或補足。
        name = require_mapping(record).get("locationName")  # 取得官方名稱。
        if (
            not isinstance(name, str) or not name.strip()
        ):  # 不用不完整資料覆蓋指令選項。
            raise WeatherError("data")  # 要求下一次啟動重新同步。
        names.add(name.strip())  # 不因該城市預報欄位缺漏而移除城市。
    if not names:  # 不建立虛假的預設城市清單。
        raise WeatherError("data")  # 明確回報清單暫缺。
    return sorted(names)  # 保留全部城市並穩定排序。


def parse_probability(value):  # 區分合法百分比與未知值。
    probability = None  # 缺漏或解析失敗維持未知。
    try:  # 數值解析集中在此函式。
        number = int(value)  # CWA 機率值為整數文字。
        if 0 <= number <= 100:  # 零是有效值。
            probability = number  # 只保留合法機率。
    except (TypeError, ValueError):  # 非整數文字不影響其他預報因子。
        pass  # 維持未知。
    return probability  # 單一回傳出口。


def element_value(entry):  # 欄位缺漏時保留其他可用預報。
    parameter = require_mapping(entry.get("parameter", {}))  # 缺漏參數視為空物件。
    value = parameter.get("parameterName")  # 讀取顯示值。
    result = value.strip() if isinstance(value, str) else ""  # 非文字值視為未知。
    return result  # 單一回傳出口。


def merge_periods(record):  # 以起訖時間合併各氣象因子，避免依賴回應順序。
    periods = {}  # 同時段的氣象資料共用一筆記錄。
    for element in require_list(
        record.get("weatherElement", [])
    ):  # 逐一讀取各氣象因子。
        element = require_mapping(element)  # 驗證因子結構。
        name = element.get("elementName")  # 取得 Wx、PoP、CI、MinT 或 MaxT。
        for entry in require_list(element.get("time", [])):  # 讀取該因子的各時段。
            entry = require_mapping(entry)  # 驗證時段結構。
            start, end = (
                entry.get("startTime"),
                entry.get("endTime"),
            )  # 共同識別預報時段。
            if (
                not isinstance(start, str)
                or not isinstance(end, str)
                or not start
                or not end
            ):  # 無起訖時間不能合併。
                raise WeatherError("data")  # 不把不明時段放入卡片。
            period = periods.setdefault(
                (start, end), {"start": start, "end": end}
            )  # 建立或取回相同時段。
            value = element_value(entry)  # 缺漏資料保留空字串。
            period[name] = (
                parse_probability(value) if name == "PoP" else value
            )  # 降雨轉成數字，其餘保持文字。
    if not periods:  # 目標城市暫時沒有預報。
        raise WeatherError("data")  # 區分地區不存在與無時段。
    return [
        periods[key] for key in sorted(periods)
    ]  # 單一回傳出口，按開始與結束時間排序。


def forecast_from_records(records, location):  # 即使 CWA 回傳多個城市，也必須明確匹配。
    matches = [
        record
        for record in records
        if require_mapping(record).get("locationName") == location
    ]  # 不直接使用第一筆。
    if not matches:  # 找不到使用者選定城市。
        raise WeatherError("location")  # 不回退成其他城市。
    return merge_periods(matches[0])  # 單一回傳出口。


def format_time(value):  # 同時支援 CWA 常見時間表示。
    result = value  # 無法辨識時保留上游文字。
    try:  # ISO 解析同時接受空格及 T 分隔。
        result = datetime.datetime.fromisoformat(
            value.replace("Z", "+00:00")
            ).strftime("%m/%d %H:%M")  # 呈現適合 Discord 閱讀的時間。
    except ValueError:  # 未知格式不影響其他欄位。
        pass  # 保留原時間文字。
    return result  # 單一回傳出口。


def rain_text(probability):  # 零值與未知值必須分開。
    return "未知" if probability is None else str(probability) + "%"  # 單一回傳出口。


def rain_summary(periods):  # 文字與卡片共用最高降雨摘要。
    available = [
        period for period in periods if period.get("PoP") is not None
    ]  # 排除缺少機率的時段。
    summary = "目前沒有可用的降雨機率資料。"  # 全部未知時的提示。
    if available:  # 只有可用資料才建立口語建議。
        highest = max(
            available, key=lambda period: period["PoP"]
        )  # 相同值保留已排序的較早時段。
        messages = next(
            messages for limit,
            messages in RAIN_MESSAGES if highest["PoP"] <= limit
        )  # 選擇對應機率區間。
        summary = "最高降雨機率：{}%\n時段：{}～{}\n{}".format(
            highest["PoP"],
            format_time(highest["start"]),
            format_time(highest["end"]),
            random.choice(messages),
        )  # 沿用隨機口語方式。
    return summary  # 單一回傳出口。


def format_forecast(location, periods, mode):  # 同時產生舊文字回覆與新的 Embed 卡片。
    rain_only = mode == "rain"  # 判斷呈現重點。
    title = "{} {}未來 36 小時{}".format(
        "🌧️" if rain_only else "🌤️", location, "降雨機率" if rain_only else "天氣"
    )  # 顯示實際選定城市。
    summary = (
        rain_summary(periods) if rain_only else ""
    )  # 同一查詢的文字與卡片共用一句建議。
    lines = [title, summary] if rain_only else [title]  # 保留原有文字功能。
    fields = []  # Discord Embed 的各預報時段。
    for period in periods:  # 依預報順序建立內容。
        time_range = (
            format_time(period["start"]) + "～" + format_time(period["end"])
        )  # 合併時段標題。
        value = "☔ 降雨機率：" + rain_text(period.get("PoP"))  # 預設降雨模式內容。
        if not rain_only:  # 完整模式另外顯示氣溫、現象與舒適度。
            value = "{}\n🌡️ {}～{}°C\n☔ 降雨 {}\n{}".format(
                period.get("Wx") or "未知",
                period.get("MinT") or "未知",
                period.get("MaxT") or "未知",
                rain_text(period.get("PoP")),
                period.get("CI") or "未知",
            )  # 每個缺值獨立顯示未知。
        fields.append(
            {"name": time_range, "value": value, "inline": True}
        )  # 桌面版並排，行動版依寬度重排。
        lines.append("\n" + time_range + "\n" + value)  # 相同預報也提供純文字形式。
    lines.append("\n" + SOURCE)  # 文字回覆標示資料來源。
    embed = {
        "title": title,
        "description": summary,
        "color": 0x3498DB,
        "fields": fields,
        "footer": {"text": SOURCE},
    }  # Python 負責卡片內容，Go 負責發送。
    if len(fields) > 25 or any(
        len(field["name"]) > 256
        or len(field["value"]) > 1024
        for field in fields
    ):  # 不送出超過 Discord 限制的異常上游資料。
        raise WeatherError("data")  # 以資料異常回覆，避免 Discord 無法更新。
    total = (
        len(title)
        + len(summary)
        + len(SOURCE)
        + sum(len(field["name"]) + len(field["value"]) for field in fields)
    )  # 檢查 Embed 總字數。
    if (
        total > 6000 or len(title) > 256 or len(summary) > 4096
    ):  # 遵守 Discord 卡片限制。
        raise WeatherError("data")  # 不截斷預報後假裝完整。
    return {"message": "\n".join(lines), "embed": embed}  # 單一回傳出口。


def execute(arguments):  # 一次命令只執行一個操作。
    result = {}  # 集中回傳資料。
    if arguments == ["locations"]:  # 啟動時同步城市清單。
        result = {
            "locations": catalog_from_records(fetch_cwa())
        }  # 不傳城市篩選且保留全部名稱。
    elif (
        len(arguments) == 3
        and arguments[0] == "forecast"
        and arguments[2] in ("weather", "rain")
    ):  # 收到 Discord 預報事件。
        location = normalize_location(arguments[1])  # 不依 ENV 偷換明確輸入城市。
        result = execute_forecast(location, arguments[2])  # 交由獨立預報操作處理。
    else:  # 未知或不完整的操作。
        raise WeatherError("input")  # 固定格式錯誤代碼。
    return result  # 單一回傳出口。


def execute_forecast(location, mode):  # 預報必須有一個完整城市名稱。
    if (
        not location
        or "," in location
        or "，" in location
        or len(location.split()) != 1
    ):  # 防止多城市與空白查詢。
        raise WeatherError("location")  # 不將無效輸入交給全部城市查詢。
    periods = forecast_from_records(
        fetch_cwa(location), location
    )  # 上游結果必須匹配本次城市。
    return format_forecast(location, periods, mode)  # 單一回傳出口。


def main(arguments=None):  # 捕捉本次操作失敗並只輸出一份 JSON。
    arguments = sys.argv[1:] if arguments is None else arguments  # 測試可注入命令參數。
    exit_code = 0  # 成功正常結束子程序。
    try:  # 所有錯誤在此集中轉成固定代碼。
        result = execute(arguments)  # 執行單次城市或預報查詢。
    except WeatherError as error:  # 已分類的設定、資料或輸入錯誤。
        result = {"error_code": str(error)}  # 不包含原始網路請求。
    except urllib.error.HTTPError as error:  # HTTPError 必須先於 URLError 處理。
        result = {
            "error_code": {401: "auth", 403: "auth", 429: "rate"}.get(
                error.code, "service"
            )
        }  # 不輸出 HTTP 本文。
    except (TimeoutError, socket.timeout):  # 查詢等待超時。
        result = {"error_code": "timeout"}  # Go 顯示可重試提示。
    except urllib.error.URLError as error:  # 網路連線或包裝後的逾時。
        result = {
            "error_code": "timeout"
            if isinstance(error.reason, (TimeoutError, socket.timeout))
            else "service"
        }  # 不輸出網址或金鑰。
    except (ValueError, TypeError, KeyError):  # JSON 與資料型別異常。
        result = {"error_code": "data"}  # 不把解析失敗當成無此城市。
    except Exception:  # 子程序的非預期失敗仍須回傳合法 JSON。
        result = {"error_code": "runtime"}  # 不把 traceback 或敏感內容送回 Discord。
    if "error_code" in result:  # 有錯誤代碼時以失敗狀態結束。
        exit_code = 1  # Go 仍可讀取 JSON 內的具體錯誤分類。
    print(json.dumps(result, ensure_ascii=False))  # stdout 只輸出一份 UTF-8 JSON。
    return exit_code  # 單一回傳出口，不建立常駐迴圈。


if __name__ == "__main__":  # 僅命令列執行時啟動，匯入測試不觸發網路查詢。
    reconfigure = getattr(sys.stdout, "reconfigure", None)
    if callable(reconfigure):
        reconfigure(encoding="utf-8")  # Windows 同樣固定使用 UTF-8 輸出。
    sys.exit(main())  # 查詢完成或失敗後立即結束程序。
