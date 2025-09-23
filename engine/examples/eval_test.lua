local eval = arboreal.llm_eval({
    system = "Evaluate the following user prompt and return a string containing the subject of the query",
    annotation = "subject",
})

local line = io.read("*l")
print()

local r = eval:call({ message.new("user", line) })

print(annotation.find(r, "subject"):value())

