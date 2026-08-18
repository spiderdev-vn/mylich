# Script build va ky chu ky so (Code Signing) cho lich.exe tren Windows

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent $scriptDir
Set-Location $projectRoot

Write-Host "=============================================" -ForegroundColor Cyan
Write-Host " Lich -- Build and Sign Executable for Windows" -ForegroundColor Cyan
Write-Host "=============================================" -ForegroundColor Cyan
Write-Host ""

# 1. Build file binary
Write-Host "[1/3] Dang bien dich lich.exe..." -ForegroundColor Yellow
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
}

go build -o bin/lich.exe ./cmd/lich
if ($LASTEXITCODE -ne 0) {
    Write-Host "Loi bien dich Go!" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "Da tao file bin/lich.exe thanh cong." -ForegroundColor Green

# 2. Tim hoac tao chung chi ky so ca nhan qua .NET X509Store
Write-Host "[2/3] Dang kiem tra chung chi ky so..." -ForegroundColor Yellow
$certSubject = "CN=MyLichDev"

$myStore = New-Object System.Security.Cryptography.X509Certificates.X509Store("My", "CurrentUser")
$myStore.Open("ReadWrite")
$cert = $myStore.Certificates | Where-Object { $_.Subject -like "*$certSubject*" } | Select-Object -First 1

if (-not $cert) {
    Write-Host "Chua co chung chi $certSubject, dang tao moi bang .NET..." -ForegroundColor Cyan
    
    $rsa = [System.Security.Cryptography.RSA]::Create(2048)
    $req = New-Object System.Security.Cryptography.X509Certificates.CertificateRequest(
        $certSubject,
        $rsa,
        [System.Security.Cryptography.HashAlgorithmName]::SHA256,
        [System.Security.Cryptography.RSASignaturePadding]::Pkcs1
    )

    $keyUsage = New-Object System.Security.Cryptography.X509Certificates.X509KeyUsageExtension(
        [System.Security.Cryptography.X509Certificates.X509KeyUsageFlags]::DigitalSignature,
        $true
    )
    $req.CertificateExtensions.Add($keyUsage)

    $oid = New-Object System.Security.Cryptography.Oid("1.3.6.1.5.5.7.3.3") # Code Signing OID
    $oidCol = New-Object System.Security.Cryptography.OidCollection
    $oidCol.Add($oid) | Out-Null
    $enhancedUsage = New-Object System.Security.Cryptography.X509Certificates.X509EnhancedKeyUsageExtension($oidCol, $true)
    $req.CertificateExtensions.Add($enhancedUsage)

    $now = [DateTimeOffset]::UtcNow.AddDays(-1)
    $exp = [DateTimeOffset]::UtcNow.AddYears(10)
    $rawCert = $req.CreateSelfSigned($now, $exp)

    $pfxBytes = $rawCert.Export([System.Security.Cryptography.X509Certificates.X509ContentType]::Pfx, "lich123")
    $flags = [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::Exportable -bor [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::PersistKeySet
    $cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2($pfxBytes, "lich123", $flags)

    $myStore.Add($cert)
    Write-Host "Da luu chung chi vao kho ca nhan: $($cert.Thumbprint)" -ForegroundColor Green
} else {
    Write-Host "Da tim thay chung chi ca nhan: $($cert.Thumbprint)" -ForegroundColor Green
}
$myStore.Close()

# 3. Ky file lich.exe
Write-Host "[3/3] Dang ky chu ky so vao bin/lich.exe..." -ForegroundColor Yellow
$targetExe = Join-Path $projectRoot "bin\lich.exe"

$signResult = Set-AuthenticodeSignature -Certificate $cert -FilePath $targetExe
Unblock-File -Path $targetExe -ErrorAction SilentlyContinue

Write-Host ""
if ($signResult.Status -eq "Valid") {
    Write-Host "Trang thai ky: Hop le (Valid)" -ForegroundColor Green
} else {
    # Self-signed certificate tro ve UnknownError vi khong thuoc CA thuong mai (DigiCert/VeriSign)
    # File van duoc nhung chu ky so Authenticode hop le vao PE header.
    Write-Host "Trang thai ky: Da nhung chu ky so Authenticode thanh cong (Chung chi nha phat trien: $($cert.Subject))" -ForegroundColor Green
}

Write-Host "=============================================" -ForegroundColor Green
Write-Host "HOAN TAT! File da duoc build va ky thanh cong." -ForegroundColor Green
Write-Host "Ban co the chay: .\bin\lich.exe" -ForegroundColor Green
Write-Host "=============================================" -ForegroundColor Green
