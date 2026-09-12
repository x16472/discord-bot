"""依命令列參數即時查詢臺灣證券交易所，並將可直接顯示的訊息以 JSON 輸出。"""

# argparse 負責解析 Go 子程序傳入的查詢類型與股票代號。
import argparse

# csv 負責解析個股日成交資訊的下載格式。
import csv

# io 將下載到的 CSV 文字轉成可逐列讀取的資料流。
import io

# json 負責解析證交所 OpenAPI 並將結果輸出給 Go。
import json

# re 負責驗證股票代號與擷取個股名稱。
import re

# sys 提供 UTF-8 標準輸出與程式結束碼。
import sys

# urllib.error 提供 HTTP 與網路連線錯誤類型。
import urllib.error

# urllib.parse 安全組合個股查詢參數。
import urllib.parse

# urllib.request 使用 Python 標準函式庫向證交所即時取得資料。
import urllib.request

# datetime 用於產生個股月份查詢參數。
from datetime import datetime, timedelta, timezone

# Decimal 避免以浮點數計算均價時產生精度誤差。
from decimal import Decimal, InvalidOperation

# REQUEST_TIMEOUT_SECONDS 限制單次證交所 HTTP 請求等待時間。
REQUEST_TIMEOUT_SECONDS = 10
# STOCK_HISTORY_MONTHS 提供股票建議所需的近期交易資料範圍。
STOCK_HISTORY_MONTHS = 3
# TWSE_TIMEZONE 確保依臺灣時間選擇個股查詢月份。
TWSE_TIMEZONE = timezone(timedelta(hours=8))
# STOCK_NUMBER_PATTERN 限制股票代號為四至六碼英數字。
STOCK_NUMBER_PATTERN = re.compile(r"^[0-9]{4,6}[A-Za-z]?$")
# OPENAPI_BASE_URL 是證交所正式 OpenAPI 的共同路徑。
OPENAPI_BASE_URL = "https://openapi.twse.com.tw/v1/exchangeReport"
# STOCK_DAY_AVG_ALL_URL 提供上市商品日收盤價及月平均價。
STOCK_DAY_AVG_ALL_URL = f"{OPENAPI_BASE_URL}/STOCK_DAY_AVG_ALL"
# MI_INDEX_URL 提供每日收盤行情的大盤統計資訊。
MI_INDEX_URL = f"{OPENAPI_BASE_URL}/MI_INDEX"
# STOCK_DAY_URL 提供個股開高低收、成交股數及成交筆數。
STOCK_DAY_URL = "https://www.twse.com.tw/exchangeReport/STOCK_DAY"
# REQUEST_HEADERS 提供證交所辨識來源所需的 User-Agent。
REQUEST_HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/120.0.0.0 Safari/537.36"
    )
}


# fetch_bytes 收到事件後才發送一次 HTTP 請求，不使用本機快取。
def fetch_bytes(url):
    # 建立包含 User-Agent 的 HTTP 請求。
    request = urllib.request.Request(url, headers=REQUEST_HEADERS)
    # 發送請求並在區塊結束時關閉回應串流。
    with urllib.request.urlopen(request, timeout=REQUEST_TIMEOUT_SECONDS) as response:
        # 回傳完整位元內容供呼叫端依 JSON 或 CSV 解碼。
        return response.read()


# fetch_openapi_records 取得並驗證證交所 OpenAPI 的陣列回應。
def fetch_openapi_records(url):
    # utf-8-sig 可同時接受一般 UTF-8 與含 BOM 的回應。
    response_text = fetch_bytes(url).decode("utf-8-sig")
    # 將 JSON 文字轉成 Python 資料。
    records = json.loads(response_text)
    # 兩個指定端點都應回傳物件陣列。
    if not isinstance(records, list):
        raise TypeError("證交所 OpenAPI 回應格式不正確")
    # 回傳由 Python 負責後續整理的資料列。
    return records


# decode_twse_csv 將個股行情位元資料解碼為 CSV 文字。
def decode_twse_csv(content):
    # 證交所 CSV 通常使用 UTF-8 BOM，先以 utf-8-sig 解碼。
    try:
        return content.decode("utf-8-sig")
    # 部分舊資料可能使用 Big5，解碼失敗時改用 Big5。
    except UnicodeDecodeError:
        return content.decode("big5", errors="replace")


# parse_decimal 將證交所數字欄位安全轉成 Decimal。
def parse_decimal(value):
    # 移除千分位、空白與證交所可能附加的比較符號。
    normalized = str(value or "").replace(",", "").strip()
    normalized = normalized.lstrip("<>Xx")
    # 空值與雙減號代表沒有可用數字。
    if normalized in ("", "--"):
        return None
    # 無法解析的欄位交由呼叫端略過或回報。
    try:
        return Decimal(normalized)
    except InvalidOperation:
        return None


# format_decimal 將 Decimal 轉成含千分位及固定小數位的文字。
def format_decimal(value, places=2):
    # 動態產生 Decimal 量化格式。
    quantum = Decimal(1).scaleb(-places)
    # 回傳適合 Discord 閱讀的千分位格式。
    return f"{value.quantize(quantum):,.{places}f}"


# format_signed_decimal 將漲跌數字固定顯示正負號。
def format_signed_decimal(value, places=2):
    # 正數補上加號，負數保留本身符號。
    prefix = "+" if value >= 0 else ""
    # 使用共同數字格式輸出。
    return f"{prefix}{format_decimal(value, places)}"


# format_roc_date 將民國年月日緊湊格式轉成斜線日期。
def format_roc_date(value):
    # 移除來源可能帶入的斜線或空白。
    digits = re.sub(r"\D", "", str(value or ""))
    # 民國年份可能是二至三位，最後四碼固定是月與日。
    if len(digits) >= 6:
        return f"{digits[:-4]}/{digits[-4:-2]}/{digits[-2:]}"
    # 無法辨識時保留來源文字，避免誤改資料。
    return str(value or "未知日期")


# signed_market_value 依 MI_INDEX 的漲跌符號建立帶方向的數值。
def signed_market_value(direction, raw_value):
    # 先將來源數字轉成絕對值，避免符號欄位與數值重複套用。
    value = parse_decimal(raw_value)
    if value is None:
        raise ValueError("大盤漲跌欄位格式不正確")
    value = abs(value)
    # 減號代表下跌，其餘方向視為上漲或持平。
    if str(direction or "").strip() == "-":
        return -value
    # 回傳帶方向的數值。
    return value


# build_market_message 使用兩個指定 OpenAPI 產生單日大盤訊息。
def build_market_message(index_records, average_records):
    # 尋找「發行量加權股價指數」這筆大盤統計資料。
    market_record = next(
        (
            record
            for record in index_records
            if str(record.get("指數", "")).strip() == "發行量加權股價指數"
        ),
        None,
    )
    # 端點沒有目標指數時停止產生不完整訊息。
    if market_record is None:
        raise ValueError("證交所 MI_INDEX 找不到發行量加權股價指數")

    # 解析加權指數的收盤值與漲跌資料。
    closing_index = parse_decimal(market_record.get("收盤指數"))
    if closing_index is None:
        raise ValueError("證交所 MI_INDEX 的收盤指數格式不正確")
    change = signed_market_value(
        market_record.get("漲跌"),
        market_record.get("漲跌點數"),
    )
    change_percent = signed_market_value(
        market_record.get("漲跌"),
        market_record.get("漲跌百分比"),
    )
    # 以 MI_INDEX 日期作為兩個端點資料的一致性基準。
    market_date = str(market_record.get("日期", "")).strip()

    # 僅統計同交易日且具有收盤價及月平均價的上市商品。
    comparable_records = []
    for record in average_records:
        # 如果端點提供日期，就排除與大盤不同交易日的資料。
        record_date = str(record.get("Date", "")).strip()
        if market_date and record_date and record_date != market_date:
            continue
        # 將收盤價與月平均價轉成 Decimal。
        closing_price = parse_decimal(record.get("ClosingPrice"))
        monthly_average = parse_decimal(record.get("MonthlyAveragePrice"))
        # 缺少任一價格的商品不納入比較。
        if closing_price is None or monthly_average is None:
            continue
        # 保存可比較價格供後續統計。
        comparable_records.append((closing_price, monthly_average))

    # 完全沒有可比較商品時回報端點資料異常。
    if not comparable_records:
        raise ValueError("證交所 STOCK_DAY_AVG_ALL 沒有同日可比較資料")
    # 分別統計收盤價高於、低於及等於月平均價的商品數。
    above_average = sum(close > average for close, average in comparable_records)
    below_average = sum(close < average for close, average in comparable_records)
    equal_average = len(comparable_records) - above_average - below_average

    # 組合可由 main.go 直接送出的 Discord 大盤訊息。
    return "\n".join(
        (
            f"📊 單日大盤（{format_roc_date(market_date)}）",
            (
                f"加權指數：{format_decimal(closing_index)}｜"
                f"漲跌：{format_signed_decimal(change)} "
                f"（{format_signed_decimal(change_percent)}%）"
            ),
            f"上市商品：{len(comparable_records):,} 檔",
            (
                f"高於月均價：{above_average:,}｜"
                f"低於月均價：{below_average:,}｜持平：{equal_average:,}"
            ),
            "資料來源：臺灣證券交易所 OpenAPI",
        )
    )


# fetch_market_message 即時呼叫兩個指定端點並完成大盤解析。
def fetch_market_message():
    # MI_INDEX 提供加權指數與當日漲跌。
    index_records = fetch_openapi_records(MI_INDEX_URL)
    # STOCK_DAY_AVG_ALL 提供上市商品收盤價與月均價。
    average_records = fetch_openapi_records(STOCK_DAY_AVG_ALL_URL)
    # Python 負責整理成 Go 可直接轉發的訊息。
    return build_market_message(index_records, average_records)


# subtract_months 取得指定日期往前推移月份後的年月。
def subtract_months(value, months):
    # 將年月轉成連續月份索引後扣除指定數量。
    month_index = value.year * 12 + value.month - 1 - months
    # 將索引還原成西元年與月份。
    return month_index // 12, month_index % 12 + 1


# fetch_stock_month_csv 即時取得指定股票某月份的日成交 CSV。
def fetch_stock_month_csv(stock_number, year, month):
    # STOCK_DAY 接受該月份任一天，固定使用一日以表達查詢月份。
    query = urllib.parse.urlencode(
        {
            "response": "csv",
            "date": f"{year:04d}{month:02d}01",
            "stockNo": stock_number,
        }
    )
    # 組合安全編碼後的證交所網址。
    url = f"{STOCK_DAY_URL}?{query}"
    # 下載並解碼個股 CSV。
    return decode_twse_csv(fetch_bytes(url))


# parse_stock_csv 將個股 CSV 轉成名稱與行情資料列。
def parse_stock_csv(csv_text, stock_number):
    # 建立可處理引號及逗號千分位的 CSV 讀取器。
    rows = list(csv.reader(io.StringIO(csv_text)))
    # 預設名稱使用股票代號，避免標題格式變動造成空值。
    stock_name = stock_number
    # 從標題「代號 名稱 各日成交資訊」擷取公司名稱。
    title_pattern = re.compile(
        rf"(?:^|\s){re.escape(stock_number)}\s+(.+?)\s+各日成交資訊"
    )
    # 逐列尋找個股標題。
    for row in rows:
        title = " ".join(cell.strip() for cell in row if cell.strip())
        match = title_pattern.search(title)
        if match:
            stock_name = match.group(1).strip()
            break

    # 找出固定行情欄位的標題列位置。
    header_index = None
    for index, row in enumerate(rows):
        normalized = [cell.strip() for cell in row]
        if "日期" in normalized and "收盤價" in normalized and "成交股數" in normalized:
            header_index = index
            break
    # 找不到標題列代表回應不是預期的個股 CSV。
    if header_index is None:
        raise ValueError(f"證交所找不到股票代號 {stock_number} 的行情")

    # 將標題名稱映射成欄位位置，降低證交所欄位順序變更的影響。
    headers = [cell.strip() for cell in rows[header_index]]
    required_headers = (
        "日期",
        "成交股數",
        "成交筆數",
        "開盤價",
        "最高價",
        "最低價",
        "收盤價",
        "漲跌價差",
    )
    if any(header not in headers for header in required_headers):
        raise ValueError("證交所個股 CSV 缺少必要欄位")
    positions = {header: headers.index(header) for header in required_headers}

    # 解析每一個有效交易日。
    records = []
    for row in rows[header_index + 1 :]:
        # 欄位不足的說明列或空白列直接忽略。
        if len(row) < len(headers):
            continue
        # 價格欄位必須可以轉成 Decimal。
        open_price = parse_decimal(row[positions["開盤價"]])
        high_price = parse_decimal(row[positions["最高價"]])
        low_price = parse_decimal(row[positions["最低價"]])
        close_price = parse_decimal(row[positions["收盤價"]])
        change = parse_decimal(row[positions["漲跌價差"]])
        volume = parse_decimal(row[positions["成交股數"]])
        transactions = parse_decimal(row[positions["成交筆數"]])
        # 無成交價格的資料列不應成為最新行情。
        if None in (open_price, high_price, low_price, close_price, change):
            continue
        # 保存 Python 格式化及分析會使用的欄位。
        records.append(
            {
                "date": row[positions["日期"]].strip(),
                "open": open_price,
                "high": high_price,
                "low": low_price,
                "close": close_price,
                "change": change,
                "volume": int(volume or 0),
                "transactions": int(transactions or 0),
            }
        )
    # 回傳名稱與依 CSV 原始順序排列的交易資料。
    return stock_name, records


# fetch_stock_history 取得近期月份個股資料，且不在程序間保存結果。
def fetch_stock_history(stock_number, month_count, minimum_records):
    # 以臺灣當下時間作為第一個查詢月份。
    now = datetime.now(tz=TWSE_TIMEZONE)
    # 保存首次辨識到的公司名稱與跨月行情。
    stock_name = stock_number
    records = []
    # 依序查詢本月及前幾個月。
    for offset in range(month_count):
        year, month = subtract_months(now, offset)
        # 取得並解析該月個股行情；月初尚無交易資料時繼續往前找。
        try:
            csv_text = fetch_stock_month_csv(stock_number, year, month)
            parsed_name, month_records = parse_stock_csv(csv_text, stock_number)
        except ValueError:
            continue
        # 使用證交所標題提供的公司名稱。
        if parsed_name != stock_number:
            stock_name = parsed_name
        # 累積跨月分析所需資料。
        records.extend(month_records)
        # 已取得本次功能所需筆數時，不再發送額外 HTTP 請求。
        if len(records) >= minimum_records:
            break
    # 依民國日期字串排序後移除可能重複的交易日。
    unique_records = {record["date"]: record for record in records}
    sorted_records = [unique_records[key] for key in sorted(unique_records)]
    # 沒有任何交易日代表股票代號不存在或來源暫時異常。
    if not sorted_records:
        raise ValueError(f"證交所找不到股票代號 {stock_number} 的行情")
    # 回傳名稱與近期行情。
    return stock_name, sorted_records


# build_stock_price_message 產生附圖格式的個股行情訊息。
def build_stock_price_message(stock_number, stock_name, record):
    # 組合可由 main.go 直接送出的 Discord 個股訊息。
    return "\n".join(
        (
            f"📈 名稱：{stock_number}（{stock_name}）",
            f"📈 {stock_number} 個股行情（{record['date']}）",
            (
                f"收盤：{format_decimal(record['close'])} 元｜"
                f"漲跌：{format_signed_decimal(record['change'])} 元"
            ),
            (
                f"開盤：{format_decimal(record['open'])}｜"
                f"最高：{format_decimal(record['high'])}｜"
                f"最低：{format_decimal(record['low'])}"
            ),
            (f"成交股數：{record['volume']:,}｜成交筆數：{record['transactions']:,}"),
            "資料來源：臺灣證券交易所",
        )
    )


# average_closing_price 計算指定筆數的近期平均收盤價。
def average_closing_price(records, period):
    # 資料不足時不產生容易誤導的平均值。
    if len(records) < period:
        return None
    # 取最後 period 個交易日的收盤價。
    recent_prices = [record["close"] for record in records[-period:]]
    # 使用 Decimal 計算精確平均。
    return sum(recent_prices, Decimal("0")) / Decimal(period)


# build_stock_suggestion_message 依近期均線產生中性的趨勢觀察。
def build_stock_suggestion_message(stock_number, stock_name, records):
    # 股票建議至少需要二十個交易日才能比較短中期均線。
    five_day_average = average_closing_price(records, 5)
    twenty_day_average = average_closing_price(records, 20)
    if five_day_average is None or twenty_day_average is None:
        raise ValueError(f"{stock_number} 近期交易資料不足，無法產生股票建議")
    # 使用最新收盤價判斷目前與均線的相對位置。
    latest = records[-1]
    close_price = latest["close"]
    # 依均線排列產生不含買賣承諾的觀察文字。
    if close_price > five_day_average > twenty_day_average:
        observation = "短中期價格偏強，可留意量能是否延續及追價風險。"
    elif close_price < five_day_average < twenty_day_average:
        observation = "短中期價格偏弱，可先觀察止跌訊號與風險承受度。"
    else:
        observation = "目前均線方向不一致，較適合等待趨勢更明確。"
    # 組合可由 main.go 直接送出的 Discord 分析訊息。
    return "\n".join(
        (
            f"📌 {stock_number}（{stock_name}）股票觀察（{latest['date']}）",
            f"收盤：{format_decimal(close_price)} 元",
            (
                f"5 日均價：{format_decimal(five_day_average)}｜"
                f"20 日均價：{format_decimal(twenty_day_average)}"
            ),
            f"觀察：{observation}",
            "以上僅依證交所歷史行情計算，不構成投資建議。",
            "資料來源：臺灣證券交易所",
        )
    )


# fetch_stock_price_message 即時取得最新個股行情並完成格式化。
def fetch_stock_price_message(stock_number):
    # 最多往前找三個月份，但取得一筆行情後立即停止查詢。
    stock_name, records = fetch_stock_history(stock_number, STOCK_HISTORY_MONTHS, 1)
    # 使用最後一筆交易日資料建立訊息。
    return build_stock_price_message(stock_number, stock_name, records[-1])


# fetch_stock_suggestion_message 即時取得近期行情並完成分析。
def fetch_stock_suggestion_message(stock_number):
    # 查詢三個月份以涵蓋二十日均價所需資料。
    stock_name, records = fetch_stock_history(stock_number, STOCK_HISTORY_MONTHS, 20)
    # 將計算與訊息組裝全部留在 Python。
    return build_stock_suggestion_message(stock_number, stock_name, records)


# create_argument_parser 建立 Go 呼叫此程式時使用的命令列格式。
def create_argument_parser():
    # 建立命令列解析器。
    parser = argparse.ArgumentParser(
        description="即時取得臺灣證券交易所行情並輸出 JSON。"
    )
    # query_type 決定大盤、個股行情或股票建議。
    parser.add_argument("query_type", choices=("market", "price", "suggestion"))
    # stock_number 僅在個股相關查詢時需要。
    parser.add_argument("stock_number", nargs="?")
    # 回傳完成設定的解析器。
    return parser


# output_json 將固定格式 JSON 寫入標準輸出，避免混入除錯文字。
def output_json(message="", error=""):
    # Windows 主控台可能預設使用 Big5，明確固定為 Go 預期的 UTF-8。
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    # ensure_ascii=False 保留繁體中文供 Discord 直接顯示。
    json.dump({"message": message, "error": error}, sys.stdout, ensure_ascii=False)
    # 加入換行，方便直接從終端機檢查輸出。
    sys.stdout.write("\n")


# main 執行一次查詢、輸出一次 JSON，完成後立即結束。
def main():
    # 解析 Go 子程序傳入的查詢參數。
    arguments = create_argument_parser().parse_args()
    # 將股票代號統一轉為大寫並移除前後空白。
    stock_number = (arguments.stock_number or "").strip().upper()
    # 個股相關模式一定需要合法股票代號。
    if arguments.query_type in (
        "price",
        "suggestion",
    ) and not STOCK_NUMBER_PATTERN.fullmatch(stock_number):
        output_json(error="股票代號格式不正確，請使用四至六碼英數字。")
        return 1

    # 每次程序只查詢使用者本次要求的資料。
    try:
        # market 模式使用兩個指定 OpenAPI 產生單日大盤資訊。
        if arguments.query_type == "market":
            message = fetch_market_message()
        # price 模式查詢指定股票最新行情。
        elif arguments.query_type == "price":
            message = fetch_stock_price_message(stock_number)
        # suggestion 模式查詢近期資料並計算均線觀察。
        else:
            message = fetch_stock_suggestion_message(stock_number)
    # 網路、HTTP、JSON 與資料格式錯誤都轉成固定 error 欄位。
    except (
        urllib.error.URLError,
        TimeoutError,
        OSError,
        UnicodeError,
        json.JSONDecodeError,
        ValueError,
    ) as error:
        output_json(error=f"證交所資料查詢失敗：{error}")
        return 1

    # 成功時只輸出已完成格式化的 message。
    output_json(message=message)
    # 使用成功結束碼。
    return 0


# 僅在 Go 或使用者直接執行 stock.py 時啟動一次查詢。
if __name__ == "__main__":
    # 將 main 的結果設為程序結束碼。
    sys.exit(main())
