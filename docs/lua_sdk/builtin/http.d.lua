---@class http
http = {}

---发起 HTTP GET 请求，返回响应体字符串。若响应内容无法转为字符串则返回空串。
---@param url string 请求地址
---@param header? table<string, string> 可选，请求头键值对
---@return string body 响应体字符串
function http.get(url, header) return "" end

---发起 HTTP POST 请求，返回响应体字符串。若响应内容无法转为字符串则返回空串。
---默认 Content-Type 为 application/x-www-form-urlencoded，可通过 header 参数覆盖。
---@param url string 请求地址
---@param data string 请求体数据
---@param header? table<string, string> 可选，请求头键值对
---@return string body 响应体字符串
function http.post(url, data, header) return "" end
