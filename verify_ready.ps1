Write-Host "=================================="
Write-Host "PRE-PHASE 1 VERIFICATION"
Write-Host "=================================="

$errors = 0

# 1. Go
Write-Host -NoNewline "1. Go installed: "
try {
    $goVersion = & go version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host $goVersion -ForegroundColor Green
    } else {
        throw
    }
} catch {
    Write-Host "FAIL - Go not found" -ForegroundColor Red
    Write-Host "   Download from: https://golang.org/dl/"
    $errors++
}

# 2. Directory
Write-Host -NoNewline "2. Directory writable: "
try {
    New-Item -ItemType Directory -Name "test-verify" -ErrorAction Stop | Out-Null
    Remove-Item "test-verify"
    Write-Host "OK" -ForegroundColor Green
} catch {
    Write-Host "FAIL" -ForegroundColor Red
    $errors++
}

# 3. Excel file
Write-Host -NoNewline "3. Excel file exists: "
if (Test-Path "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx") {
    Write-Host "OK" -ForegroundColor Green
} else {
    Write-Host "FAIL" -ForegroundColor Red
    $errors++
}

# 4. Backup
Write-Host -NoNewline "4. Backup exists: "
if (Test-Path "Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.BACKUP.xlsx") {
    Write-Host "OK" -ForegroundColor Green
} else {
    Write-Host "FAIL" -ForegroundColor Red
    $errors++
}

# 5. Python
Write-Host -NoNewline "5. Python available: "
try {
    $pyVersion = & python --version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host $pyVersion -ForegroundColor Green
    } else {
        throw
    }
} catch {
    Write-Host "FAIL" -ForegroundColor Red
    $errors++
}

# 6. openpyxl
Write-Host -NoNewline "6. openpyxl installed: "
try {
    & python -c "import openpyxl" 2>&1 | Out-Null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "OK" -ForegroundColor Green
    } else {
        throw
    }
} catch {
    Write-Host "FAIL" -ForegroundColor Red
    $errors++
}

# 7. Internet
Write-Host -NoNewline "7. Internet access: "
try {
    $response = Invoke-WebRequest -Uri "https://proxy.golang.org" -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
    Write-Host "OK" -ForegroundColor Green
} catch {
    Write-Host "FAIL (needed for 'go get')" -ForegroundColor Red
    $errors++
}

# 8. Excel verification (if Python works)
if ($errors -le 2) {
    Write-Host -NoNewline "8. Reference sheet OK: "
    try {
        $output = & python -c @"
import openpyxl
wb = openpyxl.load_workbook('Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx')
if 'Referência de Categorias' in wb.sheetnames:
    ws = wb['Referência de Categorias']
    print(f'{ws.max_row - 4}')
else:
    print('0')
wb.close()
"@ 2>&1
        if ([int]$output -eq 76) {
            Write-Host "OK (76 subcategories)" -ForegroundColor Green
        } else {
            Write-Host "FAIL (found $output, expected 76)" -ForegroundColor Red
            $errors++
        }
    } catch {
        Write-Host "FAIL" -ForegroundColor Red
        $errors++
    }
}

Write-Host "=================================="
if ($errors -eq 0) {
    Write-Host "ALL CHECKS PASSED" -ForegroundColor Green
    Write-Host "Ready to start Phase 1 with TDD" -ForegroundColor Green
} else {
    Write-Host "FAILED: $errors checks" -ForegroundColor Red
    Write-Host "Fix issues above before proceeding"
    Write-Host ""
    Write-Host "Common fixes:"
    Write-Host "- Go not found: Install from https://golang.org/dl/"
    Write-Host "- Internet fail: Check firewall/proxy settings"
}
Write-Host "=================================="
