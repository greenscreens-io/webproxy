# Scripting Web Proxy

A simple web proxy with request / response scripting interceptor allows you to use JavaScript for HTTP request headers and HTML body responses to block access or modify web content before displaying  it in the browser.

It can be used to create scripts to block ads or to inject JavaScript code into a web page such as blocking video autoplay etc.

Proxy use MITM attack to decrypt HTTPS protected websites and then use its own auto generated digital certificate. That requires a custom root certificate installed in OS Trusted Certificate Store in order for the browser to successfully load a web page.

Double-click on **root.cer** certificate file to import into Certificate Store. When a popup window opens, select Trusted Certificate store and install certificate.

Start proxy.exe and open browser configured to use proxy.

For Chrome, use  command line
`chrome.exe" --proxy-server="localhost:8888"`

To compile code
`go build -trimpath -ldflags "-s -w" -o "proxy.exe"`
