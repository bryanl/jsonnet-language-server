// local f = import "data.jsonnet";

local o = {
    nested1: {
        nested2: {
            val: "x"
        }
    }
};

local val = o.nested1.nested2.val;

{}
