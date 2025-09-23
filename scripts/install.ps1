#!/usr/bin/env pwsh
# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0

param(
    [string]$Version,
    [switch]$Force,
    [switch]$NoSymlink
)

$ErrorActionPreference = 'Stop'
$repo = 'choreoatlas2025/cli'
$apiBase = "https://api.github.com/repos/$repo"
$downloadBase = "https://github.com/$repo/releases/download"
$temp = [System.IO.Path]::GetTempPath()
$tmpDir = [System.IO.Path]::Combine($temp, "choreoatlas-install-" + [System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmpDir | Out-Null

function Cleanup {
    if (Test-Path $tmpDir) {
        Remove-Item -Path $tmpDir -Recurse -Force
    }
}

trap {
    Cleanup
    throw
}

try {
    function Get-LatestTag {
        Write-Host "[INFO] Resolving latest release tag"
        $response = Invoke-WebRequest -Uri "$apiBase/releases/latest" -Headers @{ 'Accept' = 'application/vnd.github+json' }
        $json = $response.Content | ConvertFrom-Json
        if (-not $json.tag_name) {
            throw "Unable to determine latest release tag"
        }
        return $json.tag_name
    }

    $tag = if ([string]::IsNullOrWhiteSpace($Version)) { Get-LatestTag } else { $Version }
    if (-not $tag.StartsWith('v')) {
        $tag = "v$tag"
    }

    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        'X64'   { $arch = 'amd64' }
        'Arm64' { $arch = 'arm64' }
        default { throw "Unsupported architecture: $arch" }
    }

    $archiveName = "choreoatlas_${tag}_windows_${arch}.zip"
    $downloadUrl = "$downloadBase/$tag/$archiveName"
    $shaUrl = "$downloadBase/$tag/SHA256SUMS.txt"

    Write-Host "[INFO] Installing ChoreoAtlas $tag for windows/$arch"

    $archivePath = Join-Path $tmpDir $archiveName
    Write-Host "[INFO] Downloading $archiveName"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing

    $shaPath = Join-Path $tmpDir 'SHA256SUMS.txt'
    Write-Host "[INFO] Downloading checksums"
    Invoke-WebRequest -Uri $shaUrl -OutFile $shaPath -UseBasicParsing

    $expectedLine = (Get-Content $shaPath | Where-Object { $_ -match " $archiveName`$" })
    if (-not $expectedLine) {
        throw "Checksum for $archiveName not found"
    }
    $expectedHash = ($expectedLine -split '\s+')[0].ToLowerInvariant()
    $actualHash = (Get-FileHash -Algorithm SHA256 -Path $archivePath).Hash.ToLowerInvariant()
    if ($expectedHash -ne $actualHash) {
        throw "Checksum mismatch for $archiveName"
    }

    Write-Host "[INFO] Extracting archive"
    Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force

    $binaryPath = Join-Path $tmpDir 'choreoatlas.exe'
    if (-not (Test-Path $binaryPath)) {
        throw "Extracted binary not found"
    }

    $candidateDirs = @()
    if ($Env:ProgramFiles) {
        $candidateDirs += (Join-Path $Env:ProgramFiles 'ChoreoAtlas')
    }
    if ($Env:LOCALAPPDATA) {
        $candidateDirs += (Join-Path $Env:LOCALAPPDATA 'ChoreoAtlas\bin')
    }

    $installDir = $null
    foreach ($dir in $candidateDirs) {
        try {
            if (-not (Test-Path $dir)) {
                New-Item -ItemType Directory -Path $dir -Force | Out-Null
            }
            $testFile = Join-Path $dir '.__write_test'
            Set-Content -Path $testFile -Value 'ok'
            Remove-Item -Path $testFile -Force
            $installDir = $dir
            break
        } catch {
            continue
        }
    }

    if (-not $installDir) {
        throw "No writable install directory found. Try running in an elevated prompt or set VERSION with an existing writable path."
    }

    $targetPath = Join-Path $installDir 'choreoatlas.exe'
    Write-Host "[INFO] Installing to $targetPath"
    Copy-Item -Path $binaryPath -Destination $targetPath -Force

    if (-not $NoSymlink) {
        $linkPath = Join-Path $installDir 'ca.exe'
        if (Test-Path $linkPath) {
            if ($Force) {
                Remove-Item -Path $linkPath -Force
            } else {
                Write-Warning "Skipping ca.exe helper: $linkPath already exists. Use -Force or -NoSymlink to override."
                $linkPath = $null
            }
        }

        if ($linkPath) {
            try {
                New-Item -ItemType SymbolicLink -Path $linkPath -Target 'choreoatlas.exe' -Force | Out-Null
            } catch {
                Write-Warning "Symbolic link creation failed ($($_.Exception.Message)). Creating ca.cmd launcher instead."
                $linkPath = Join-Path $installDir 'ca.cmd'
                $launcher = "@echo off`r`n\"%~dp0choreoatlas.exe\" %*"
                if (Test-Path $linkPath -and -not $Force) {
                    Write-Warning "Skipping ca.cmd helper: $linkPath already exists. Use -Force to overwrite."
                } else {
                    Set-Content -Path $linkPath -Value $launcher -Encoding ASCII
                }
            }
        }
    }

    Write-Host "[INFO] Installation complete"
    Write-Host "       Binary : $targetPath"
    if (-not $NoSymlink) {
        if (Test-Path (Join-Path $installDir 'ca.exe')) {
            Write-Host "       Helper : $(Join-Path $installDir 'ca.exe')"
        } elseif (Test-Path (Join-Path $installDir 'ca.cmd')) {
            Write-Host "       Helper : $(Join-Path $installDir 'ca.cmd')"
        } else {
            Write-Host "       Helper : skipped"
        }
    } else {
        Write-Host "       Helper : skipped"
    }
    Write-Host "[INFO] Run 'choreoatlas version' to verify installation"
} finally {
    Cleanup
}
