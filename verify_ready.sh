#!/bin/bash

echo "=================================="
echo "PRE-PHASE 1 VERIFICATION"
echo "=================================="

errors=0

# 1. Go
echo -n "1. Go installed: "
if go version >/dev/null 2>&1; then
    go version
else
    echo "FAIL - Go not found"
    ((errors++))
fi

# 2. Directory access
echo -n "2. Directory writable: "
if mkdir test-verify 2>/dev/null && rmdir test-verify 2>/dev/null; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 3. Excel file
echo -n "3. Excel file exists: "
if [ -f "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx" ]; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 4. Backup exists
echo -n "4. Backup exists: "
if [ -f "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.BACKUP.xlsx" ]; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 5. Python
echo -n "5. Python available: "
if python --version >/dev/null 2>&1; then
    python --version
else
    echo "FAIL"
    ((errors++))
fi

# 6. openpyxl
echo -n "6. openpyxl installed: "
if python -c "import openpyxl" 2>/dev/null; then
    echo "OK"
else
    echo "FAIL"
    ((errors++))
fi

# 7. Internet
echo -n "7. Internet access: "
if curl -s -I https://proxy.golang.org >/dev/null 2>&1 || ping -n 2 proxy.golang.org >/dev/null 2>&1; then
    echo "OK"
else
    echo "FAIL (needed for 'go get')"
    ((errors++))
fi

echo "=================================="
if [ $errors -eq 0 ]; then
    echo "ALL CHECKS PASSED"
    echo "Ready to start Phase 1 with TDD"
else
    echo "FAILED: $errors checks"
    echo "Fix issues above before proceeding"
fi
echo "=================================="
