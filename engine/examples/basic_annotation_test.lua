local s = arboreal.state("foo", "bar", function(history)
    return annotation.append(history, { name="test", value="poo" })
end)

local h = s:call({ message.new("user", "hi") })

print(h[#h]:content())
print(annotation.find(h, "test"):value())