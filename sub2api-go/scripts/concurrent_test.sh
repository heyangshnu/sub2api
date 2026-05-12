#!/bin/bash
# 并发扣费测试脚本

API_KEY="$1"
CONCURRENT="${2:-10}"
BASE_URL="${3:-http://localhost:3000}"

if [ -z "$API_KEY" ]; then
    echo "Usage: $0 <api_key> [concurrent_requests] [base_url]"
    echo "Example: $0 sk-sub2api-xxx 50 http://localhost:3000"
    exit 1
fi

echo "=========================================="
echo "并发扣费测试"
echo "=========================================="
echo "API Key: ${API_KEY:0:30}..."
echo "并发数: $CONCURRENT"
echo "API URL: $BASE_URL"
echo ""

# 获取初始余额
echo "1. 获取初始余额..."
INITIAL=$(curl -s "$BASE_URL/v1/usage" -H "Authorization: Bearer $API_KEY" | grep -o '"balance":[0-9.]*' | cut -d: -f2)
echo "   初始余额: $INITIAL"
echo ""

# 并发发送请求
echo "2. 发送 $CONCURRENT 个并发请求..."
START_TIME=$(date +%s.%N)

for i in $(seq 1 $CONCURRENT); do
    curl -s -X POST "$BASE_URL/v1/chat/completions" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"model":"deepseek-chat","messages":[{"role":"user","content":"1"}],"stream":false}' \
        > /dev/null &
done

# 等待所有请求完成
wait

END_TIME=$(date +%s.%N)
DURATION=$(echo "$END_TIME - $START_TIME" | bc)
echo "   完成! 耗时: ${DURATION}s"
echo ""

# 获取最终余额
echo "3. 获取最终余额..."
sleep 1
FINAL=$(curl -s "$BASE_URL/v1/usage" -H "Authorization: Bearer $API_KEY" | grep -o '"balance":[0-9.]*' | cut -d: -f2)
USAGE=$(curl -s "$BASE_URL/v1/usage" -H "Authorization: Bearer $API_KEY")
echo "   最终余额: $FINAL"
echo "   用量详情: $USAGE"
echo ""

# 计算结果
SPENT=$(echo "$INITIAL - $FINAL" | bc)
echo "=========================================="
echo "测试结果"
echo "=========================================="
echo "初始余额: $INITIAL"
echo "最终余额: $FINAL"
echo "总消费:   $SPENT"
echo ""

# 检查是否有超扣
if (( $(echo "$FINAL < 0" | bc -l) )); then
    echo "❌ 测试失败: 检测到超扣! 余额为负"
    exit 1
else
    echo "✅ 测试通过: 无超扣发生"
fi
