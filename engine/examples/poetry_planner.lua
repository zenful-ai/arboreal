local haiku_bot = arboreal.tree("haiku_bot", "Write a haiku for the user", "Write me a haiku about springtime")

haiku_bot:add(arboreal.llm_complete({system="Write a haiku to the user's requirements"}))

local planner = arboreal.planner("Steve", "Steve only writes poetry?", haiku_bot)

planner:oob(arboreal.state("a", "a", function(history)
    table.insert(history, message.new("assistant", "Out of bounds, baby!"))
    return history, nil
end))

arboreal.entry = planner