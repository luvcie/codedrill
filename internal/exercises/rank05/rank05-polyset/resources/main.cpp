#include "searchable_array_bag.hpp"
#include "searchable_tree_bag.hpp"
#include "set.hpp"
#include <cassert>
#include <iostream>
#include <sstream>

template<class Container> static std::string printed(const Container& value) {
    std::ostringstream out;
    std::streambuf* previous = std::cout.rdbuf(out.rdbuf());
    value.print();
    std::cout.rdbuf(previous);
    return out.str();
}

template<class Container> static void check(const std::string& bag_order, const std::string& set_order) {
    Container original;
    assert(!original.has(0));
    int input[] = {7, -4, 0, 7};
    original.insert(input, 4);
    assert(printed(original) == bag_order);
    const Container copy(original);
    Container assigned;
    assigned.insert(99);
    assert(&(assigned = original) == &assigned);
    Container& self = assigned;
    assigned = self;
    original.clear();
    assert(!original.has(7) && printed(original) == "\n");
    assert(copy.has(-4) && copy.has(0) && copy.has(7) && !copy.has(6));
    assert(printed(assigned) == bag_order);
    assigned.clear();
    assert(copy.has(7));
    {
        set values(original);
        int repeated[] = {2, 2, -1, 0, 2};
        values.insert(repeated, 5);
        values.insert(-1);
        const set& view = values;
        assert(view.has(2) && view.has(-1) && view.has(0) && !view.has(7));
        assert(&view.get_bag() == static_cast<const searchable_bag*>(&original));
        const std::string result = printed(values);
        assert(result == printed(original));
        assert(result == set_order);
        values.clear();
        assert(!values.has(2) && printed(values) == "\n");
        values.insert(13);
    }
    assert(original.has(13));
    Container scoped(copy);
    searchable_bag& polymorphic = scoped;
    polymorphic.insert(100);
    assert(polymorphic.has(100) && !copy.has(100));
}

int main() {
    check<searchable_array_bag>("7 -4 0 7 \n", "2 -1 0 \n");
    check<searchable_tree_bag>("-4 7 \n", "-1 2 \n");
    searchable_tree_bag source, destination;
    source.insert(11);
    destination.insert(8);
    destination.set_tree(source.extract_tree());
    assert(!source.has(11) && destination.has(11) && !destination.has(8));

    std::ostringstream diagnostics;
    std::streambuf* previous = std::cout.rdbuf(diagnostics.rdbuf());
    {
        tree_bag fixture;
        fixture.insert(0);
        fixture.insert(5);
        fixture.insert(0);
        fixture.print();
        fixture.clear();
    }
    std::cout.rdbuf(previous);
    assert(diagnostics.str() == "create node: 0\ncreate node: 5\ncreate node: 0\n"
           "duplicate value: delete node\n5 \ndestroying value: 0\ndestroying value: 5\n");
}
