local tree = arboreal.tree("chat", "chat with the user", "hi there")

local llm = arboreal.llm_complete({system="Respond only in Haikus"})

tree:add(llm)

arboreal.entry = tree