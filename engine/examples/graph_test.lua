local tree = arboreal.tree("test", "foo", "bar")

local a = arboreal.state("a", "a", function(history)
	print("a")
	return history, nil
end)

local b = arboreal.state("b", "b", function(history)
	print("b")
    return history, signal.user("I need input!")
	--return history, nil
end)

local c = arboreal.state("c", "c", function(history)
	print("c")
	return history, nil
end)

local d = arboreal.state("d", "d", function(history)
	print("d")
	return history, nil
end)

tree:add(a, b)
tree:add(a, c)
tree:add(b, d)

arboreal.entry = tree
