# ==========================================================================
# Cozy Canvas - Lightweight PowerShell Web Server
# Serves frontend static files without Node.js dependencies
# ==========================================================================

$port = 8080
$listener = New-Object System.Net.HttpListener
$listener.Prefixes.Add("http://localhost:$port/")

try {
    $listener.Start()
} catch {
    Write-Error "Failed to start listener on port $port. It might already be in use."
    exit
}

Write-Host "==================================================" -ForegroundColor Green
Write-Host "*** Cozy Canvas static web server is running! ***" -ForegroundColor Cyan
Write-Host "Open: http://localhost:$port/" -ForegroundColor Yellow
Write-Host "Press Ctrl+C in this window to stop the server." -ForegroundColor White
Write-Host "==================================================" -ForegroundColor Green

try {
    while ($listener.IsListening) {
        $context = $listener.GetContext()
        $request = $context.Request
        $response = $context.Response

        $urlPath = $request.Url.LocalPath
        if ($urlPath -eq "/") {
            $urlPath = "/index.html"
        }

        # Safe directory resolution to serve files relative to script directory
        $scriptPath = Split-Path -Parent $MyInvocation.MyCommand.Path
        if ($scriptPath -eq $null -or $scriptPath -eq "") {
            $scriptPath = Get-Location
        }
        $localPath = [System.IO.Path]::GetFullPath([System.IO.Path]::Combine($scriptPath, $urlPath.TrimStart([char]47)))

        if (Test-Path $localPath -PathType Leaf) {
            $bytes = [System.IO.File]::ReadAllBytes($localPath)
            
            # Content Type Mapping
            if ($localPath.EndsWith(".html")) {
                $response.ContentType = "text/html; charset=utf-8"
            }
            elseif ($localPath.EndsWith(".css")) {
                $response.ContentType = "text/css; charset=utf-8"
            }
            elseif ($localPath.EndsWith(".js")) {
                $response.ContentType = "application/javascript; charset=utf-8"
            }
            elseif ($localPath.EndsWith(".svg")) {
                $response.ContentType = "image/svg+xml; charset=utf-8"
            }
            elseif ($localPath.EndsWith(".json")) {
                $response.ContentType = "application/json; charset=utf-8"
            }
            
            $response.ContentLength64 = $bytes.Length
            $response.OutputStream.Write($bytes, 0, $bytes.Length)
        } else {
            $response.StatusCode = 404
            $errorMessage = "404 - Cozy File Not Found: $urlPath"
            $errBytes = [System.Text.Encoding]::UTF8.GetBytes($errorMessage)
            $response.ContentLength64 = $errBytes.Length
            $response.OutputStream.Write($errBytes, 0, $errBytes.Length)
        }
        $response.Close()
    }
}
catch {
    Write-Host "`nStopping server..." -ForegroundColor Yellow
}
finally {
    $listener.Stop()
}
