#!/bin/bash
# ScrapeGoat Test Runner & Benchmark Suite
# Usage: ./scripts/test.sh [unit|integration|bench|load|all]

set -e

CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

echo -e "${CYAN}═══════════════════════════════════════════${NC}"
echo -e "${CYAN}  🕸️  ScrapeGoat Test Suite${NC}"
echo -e "${CYAN}═══════════════════════════════════════════${NC}"

run_unit_tests() {
    echo -e "\n${YELLOW}▸ Running Unit Tests${NC}"
    echo "────────────────────────────────────"
    go test -v -short -count=1 ./internal/engine/ ./internal/parser/ ./internal/pipeline/ 2>&1 | while IFS= read -r line; do
        if echo "$line" | grep -q "^--- PASS"; then
            echo -e "${GREEN}$line${NC}"
        elif echo "$line" | grep -q "^--- FAIL"; then
            echo -e "${RED}$line${NC}"
        else
            echo "$line"
        fi
    done
    echo -e "${GREEN}✓ Unit tests complete${NC}\n"
}

run_integration_tests() {
    echo -e "\n${YELLOW}▸ Running Integration Tests (live network)${NC}"
    echo "────────────────────────────────────"
    go test -v -count=1 -timeout 120s ./tests/ 2>&1 | while IFS= read -r line; do
        if echo "$line" | grep -q "^--- PASS"; then
            echo -e "${GREEN}$line${NC}"
        elif echo "$line" | grep -q "^--- FAIL"; then
            echo -e "${RED}$line${NC}"
        else
            echo "$line"
        fi
    done
    echo -e "${GREEN}✓ Integration tests complete${NC}\n"
}

run_benchmarks() {
    echo -e "\n${YELLOW}▸ Running Benchmarks${NC}"
    echo "────────────────────────────────────"
    
    echo -e "\n${CYAN}  Parser Benchmarks:${NC}"
    go test -bench=. -benchmem -benchtime=3s -short ./internal/parser/ 2>&1 | grep -E "Benchmark|PASS|FAIL"

    echo -e "\n${CYAN}  Pipeline Benchmarks:${NC}"
    go test -bench=. -benchmem -benchtime=3s -short ./internal/pipeline/ 2>&1 | grep -E "Benchmark|PASS|FAIL"

    echo -e "\n${CYAN}  Engine Benchmarks:${NC}"
    go test -bench=. -benchmem -benchtime=3s -short ./internal/engine/ 2>&1 | grep -E "Benchmark|PASS|FAIL"

    echo -e "${GREEN}✓ Benchmarks complete${NC}\n"
}

run_load_test() {
    echo -e "\n${YELLOW}▸ Running Load Test (API Stress Test via vegeta)${NC}"
    echo "────────────────────────────────────"

    # Build and start the API server in background
    make build 2>/dev/null
    ./bin/scrapegoat crawl https://quotes.toscrape.com --depth 0 --concurrency 1 &
    CRAWL_PID=$!
    sleep 3

    # Check if metrics endpoint is available
    if ! curl -s http://localhost:9090/health > /dev/null 2>&1; then
        echo -e "${YELLOW}  Metrics server not available, using API test target instead${NC}"
    fi

    # Test: Sustained load on metrics endpoint
    echo -e "\n${CYAN}  Test 1: Metrics endpoint — 200 req/s for 10s${NC}"
    echo "GET http://localhost:9090/metrics" | vegeta attack -rate=200/s -duration=10s | vegeta report 2>/dev/null || echo "  (vegeta not found or endpoint unavailable)"

    echo -e "\n${CYAN}  Test 2: Health endpoint — 500 req/s for 10s${NC}"
    echo "GET http://localhost:9090/health" | vegeta attack -rate=500/s -duration=10s | vegeta report 2>/dev/null || echo "  (vegeta not found or endpoint unavailable)"

    # Cleanup
    kill $CRAWL_PID 2>/dev/null
    wait $CRAWL_PID 2>/dev/null

    echo -e "${GREEN}✓ Load tests complete${NC}\n"
}

run_crawl_examples() {
    echo -e "\n${YELLOW}▸ Live Crawl Examples${NC}"
    echo "────────────────────────────────────"
    make build 2>/dev/null

    echo -e "\n${CYAN}  Example 1: Single page fetch (depth 0)${NC}"
    rm -rf /tmp/scrapegoat_test1
    timeout 30 ./bin/scrapegoat crawl https://books.toscrape.com --depth 0 --concurrency 1 --format json --output /tmp/scrapegoat_test1 2>&1
    echo -e "  Output: $(wc -l < /tmp/scrapegoat_test1/results.json 2>/dev/null || echo 'N/A') lines"

    echo -e "\n${CYAN}  Example 2: Shallow crawl — quotes.toscrape.com${NC}"
    rm -rf /tmp/scrapegoat_test2
    timeout 45 ./bin/scrapegoat crawl https://quotes.toscrape.com --depth 1 --concurrency 3 --format jsonl --output /tmp/scrapegoat_test2 2>&1
    echo -e "  Output: $(wc -l < /tmp/scrapegoat_test2/results.jsonl 2>/dev/null || echo 'N/A') lines"

    echo -e "\n${CYAN}  Example 3: CSV output${NC}"
    rm -rf /tmp/scrapegoat_test3
    timeout 30 ./bin/scrapegoat crawl https://books.toscrape.com --depth 0 --concurrency 1 --format csv --output /tmp/scrapegoat_test3 2>&1
    echo -e "  Output: $(wc -l < /tmp/scrapegoat_test3/results.csv 2>/dev/null || echo 'N/A') lines"

    echo -e "${GREEN}✓ Crawl examples complete${NC}\n"
}

# Argument handling
case "${1:-all}" in
    unit)
        run_unit_tests
        ;;
    integration)
        run_integration_tests
        ;;
    bench)
        run_benchmarks
        ;;
    load)
        run_load_test
        ;;
    crawl)
        run_crawl_examples
        ;;
    all)
        run_unit_tests
        run_benchmarks
        run_integration_tests
        run_crawl_examples
        run_load_test
        ;;
    *)
        echo "Usage: $0 [unit|integration|bench|load|crawl|all]"
        exit 1
        ;;
esac

echo -e "${CYAN}═══════════════════════════════════════════${NC}"
echo -e "${GREEN}  ✅ All tests complete!${NC}"
echo -e "${CYAN}═══════════════════════════════════════════${NC}"
