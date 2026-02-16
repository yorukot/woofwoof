# woofwoof

把一般文字轉成「狗語」，也可以再解碼回原文的 Go CLI。

![Demo](assets/demo.jpg)

## Quick Start

```bash
# 直接用 root command（預設 encode）
go run . "你好"

# 或明確指定 mode
go run . --mode encode "你好"

# 解碼
go run . --mode decode "汪 汪 汪 汪 汪. 嗷汪！ 嗷~ ~汪! 嗷汪 嗚！ 嗷！ 嗚汪! 嗷汪~. 嗷"
```

## Usage

```bash
# 1) root command + --mode / -m
woofwoof --mode encode "我是小狗"
woofwoof -m decode "汪 汪 汪 汪 汪～ 嗚！ 汪汪~ 嗚 嗚汪… 汪汪！ 汪汪~ 汪汪 嗷汪～ ~汪！ 嗷！ 汪嗚 嗚汪～ ~汪！ 汪汪！ 嗚～ 嗚汪! 汪嗚"

# 2) subcommand
woofwoof encode "我是小狗"
woofwoof decode "汪 汪 汪 汪 汪～ 嗚！ 汪汪~ 嗚 嗚汪… 汪汪！ 汪汪~ 汪汪 嗷汪～ ~汪！ 嗷！ 汪嗚 嗚汪～ ~汪！ 汪汪！ 嗚～ 嗚汪! 汪嗚"

# 3) 從 stdin 讀取
printf "我是小狗" | woofwoof encode
printf "汪 汪 汪 汪 汪～ 嗚！ 汪汪~ 嗚 嗚汪… 汪汪！ 汪汪~ 汪汪 嗷汪～ ~汪！ 嗷！ 汪嗚 嗚汪～ ~汪！ 汪汪！ 嗚～ 嗚汪! 汪嗚" | woofwoof decode
```

## Build

```bash
go build -o woofwoof .
```

## Notes

- 支援 UTF-8 文字（含中文）。
- 輸入可用參數或 stdin（未提供參數時會讀 stdin）。
- `--mode` 可用 `encode|enc` 或 `decode|dec`，預設是 `encode`。
- 解碼輸入必須是以空白分隔的狗語 token。
- 若 token 非法、資料不完整或內容不是有效 UTF-8，會回傳錯誤。
