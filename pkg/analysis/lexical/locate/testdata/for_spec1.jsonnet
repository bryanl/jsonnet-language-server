local names = ["a", "b", "c"];
local fn(n) = "Hello " + n;

[
    fn(n)
    for n in names
]