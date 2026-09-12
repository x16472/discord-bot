"""依命令列參數即時查詢臺灣證券交易所，並將結果以 JSON 輸出。"""

# argparse 負責解析 Go 子程序傳入的查詢類型與股票代號。
import argparse

# json 負責將查詢結果輸出給 Go 程式。
import json

# re 負責驗證股票代號格式。
import re

# sys 提供標準輸出與程式結束碼。
import sys

# urllib.error 提供 HTTP 與網路連線錯誤類型。
import urllib.error

# urllib.request 使用 Python 標準函式庫向證交所取得資料。
import urllib.request

# datetime 用於指定查詢月份及尋找最近交易日。
from datetime import datetime, timedelta

# REQUEST_TIMEOUT_SECONDS 限制單次證交所 HTTP 請求等待時間。
REQUEST_TIMEOUT_SECONDS = 10
# MARKET_LOOKBACK_DAYS 涵蓋週末與連續休市期間。
MARKET_LOOKBACK_DAYS = 10
# STOCK_NUMBER_PATTERN 限制股票代號為四至六碼英數字。
STOCK_NUMBER_PATTERN = re.compile(r"^[0-9A-Za-z]{4,6}$")
# REQUEST_HEADERS 提供證交所辨識來源所需的 User-Agent。
REQUEST_HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/120.0.0.0 Safari/537.36"
    )
}


# decode_twse_csv 將證交所回應位元資料解碼為可供 Go 解析的 CSV 字串。
def decode_twse_csv(content):
    # 證交所 CSV 通常使用 UTF-8 BOM，先以 utf-8-sig 解碼。
    try:
        return content.decode("utf-8-sig")
    # 部分舊資料可能使用 Big5，解碼失敗時改用 Big5。
    except UnicodeDecodeError:
        return content.decode("big5", errors="replace")


# fetch_twse_csv 使用 Python 標準函式庫即時取得指定證交所 CSV。
def fetch_twse_csv(url):
    # 建立包含 User-Agent 的 HTTP 請求。
    request = urllib.request.Request(url, headers=REQUEST_HEADERS)
    # 發送請求並在區塊結束時關閉回應串流。
    with urllib.request.urlopen(
        request,
        timeout=REQUEST_TIMEOUT_SECONDS,
    ) as response:
        # 讀取完整回應後轉為 CSV 文字。
        return decode_twse_csv(response.read())


# fetch_latest_market_csv 逐日向前尋找最近可用的單日大盤資料。
def fetch_latest_market_csv():
    # 最多向前查詢十天，以處理週末與連續休市日。
    for days_ago in range(MARKET_LOOKBACK_DAYS):
        # 計算本次準備查詢的日期。
        query_date = datetime.now() - timedelta(days=days_ago)
        # 將日期轉成證交所 API 接受的西元年月日格式。
        date_string = query_date.strftime("%Y%m%d")
        # 組合證交所單日市場行情 CSV 網址。
        url = (
            "https://www.twse.com.tw/exchangeReport/MI_INDEX"
            f"?response=csv&date={date_string.replace('-', '')}&type=ALLBUT0999"
        )
        # 收到 Go 指令後才執行本次 HTTP 請求，不使用快取。
        csv_text = fetch_twse_csv(url)
        # 確認回應包含主要大盤指標，避免回傳休市提示頁。
        if "發行量加權股價指數" in csv_text:
            return date_string, csv_text
    # 查詢範圍內都沒有交易資料時回報明確原因。
    raise ValueError("最近十天找不到可用的單日大盤資料")


# fetch_single_stock_csv 查詢指定股票於當月截至目前的每日行情。
def fetch_single_stock_csv(stock_number):
    # STOCK_DAY 使用指定月份回傳當月所有可用交易日資料。
    date_string = datetime.now().strftime("%Y%m%d")
    # 組合證交所個股日成交資訊 CSV 網址。
    url = (
        "https://www.twse.com.tw/exchangeReport/STOCK_DAY"
        f"?response=csv&date={date_string}&stockNo={stock_number}"
    )
    # 收到 Go 指令後才執行本次 HTTP 請求，不使用快取。
    csv_text = fetch_twse_csv(url)
    # 確認回應確實包含個股行情欄位。
    if "日期" not in csv_text or "收盤價" not in csv_text:
        raise ValueError(f"證交所找不到股票代號 {stock_number} 的行情")
    # 回傳查詢日期與原始 CSV，交由 Go 統一格式化 Discord 訊息。
    return date_string, csv_text


# create_argument_parser 建立 Go 呼叫此程式時使用的命令列格式。
def create_argument_parser():
    # 建立命令列解析器。
    parser = argparse.ArgumentParser(
        description="即時取得臺灣證券交易所行情並輸出 JSON。"
    )
    # query_type 決定查詢單日大盤或指定個股。
    parser.add_argument("query_type", choices=("market", "stock"))
    # stock_number 僅在個股查詢時需要。
    parser.add_argument("stock_number", nargs="?")
    # 回傳完成設定的解析器。
    return parser


# build_success_result 建立欄位固定的成功 JSON 資料。
def build_success_result(date_string, all_market_raw="", single_stock_raw=""):
    # 固定欄位可讓 Go 使用單一結構解析大盤與個股結果。
    return {
        "date": date_string,
        "all_market_raw": all_market_raw,
        "single_stock_raw": single_stock_raw,
        "error": "",
    }


# output_json 將單一 JSON 物件寫入標準輸出，避免混入除錯文字。
def output_json(result):
    # Windows 主控台可能預設使用 Big5，明確固定為 Go 預期的 UTF-8。
    if hasattr(sys.stdout, "reconfigure"):
        sys.stdout.reconfigure(encoding="utf-8")
    # ensure_ascii=False 保留繁體中文，Go 可直接顯示錯誤內容。
    json.dump(result, sys.stdout, ensure_ascii=False)
    # 加入換行，方便直接從終端機檢查輸出。
    sys.stdout.write("\n")


# main 執行一次查詢、輸出一次 JSON，完成後立即結束。
def main():
    # 解析 Go 子程序傳入的查詢參數。
    arguments = create_argument_parser().parse_args()
    # 將股票代號統一轉為大寫並移除前後空白。
    stock_number = (arguments.stock_number or "").strip().upper()

    # 驗證個股模式一定提供合法股票代號。
    if arguments.query_type == "stock" and not STOCK_NUMBER_PATTERN.fullmatch(
        stock_number
    ):
        output_json(
            {
                "date": "",
                "all_market_raw": "",
                "single_stock_raw": "",
                "error": "股票代號格式不正確，請使用四至六碼英數字。",
            }
        )
        return 1

    # 每次程式執行只查詢使用者本次要求的資料。
    try:
        # market 模式查詢最近可用的單日大盤 CSV。
        if arguments.query_type == "market":
            date_string, csv_text = fetch_latest_market_csv()
            result = build_success_result(
                date_string,
                all_market_raw=csv_text,
            )
        # stock 模式查詢指定股票的當月日成交 CSV。
        else:
            date_string, csv_text = fetch_single_stock_csv(stock_number)
            result = build_success_result(
                date_string,
                single_stock_raw=csv_text,
            )
    # urllib 網路、HTTP 或逾時例外會轉成固定 JSON 錯誤欄位。
    except (urllib.error.URLError, TimeoutError, OSError) as error:
        result = {
            "date": "",
            "all_market_raw": "",
            "single_stock_raw": "",
            "error": f"證交所連線失敗：{error}",
        }
    # 找不到有效市場資料時同樣透過 JSON 回報。
    except ValueError as error:
        result = {
            "date": "",
            "all_market_raw": "",
            "single_stock_raw": "",
            "error": str(error),
        }

    # 將本次查詢結果輸出給 Go 子程序讀取。
    output_json(result)
    # error 欄位有內容時使用失敗結束碼，否則回傳成功。
    return 1 if result["error"] else 0


# 僅在 Go 或使用者直接執行 stock.py 時啟動一次查詢。
if __name__ == "__main__":
    # 將 main 的結果設為程序結束碼。
    sys.exit(main())
