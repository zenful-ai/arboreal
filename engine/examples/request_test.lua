local http = require("http")
local json = require("json")

response, error = http.request("GET", "http://localhost:8080/api/v1/indices")

local data = json.decode(response.body)

print(response.body)
for k, v in pairs(data) do
    print(v.id)
end


arboreal.entry = arboreal.state("foo", "bar", function(history)

    return history, nil
end)