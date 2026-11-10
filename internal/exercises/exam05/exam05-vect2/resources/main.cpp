#include "vect2.hpp"
#include <cassert>
#include <sstream>

int main() {
    assert(vect2() == vect2(0, 0));
    for (int x = -3; x <= 3; ++x) {
        for (int y = -2; y <= 2; ++y) {
            const vect2 original(x, y);
            assert(original[0] == x && original[1] == y);
            assert(original == vect2(x, y));
            assert(original != vect2(x + 1, y));
            assert(original != vect2(x, y + 1));
            vect2 value(original), assigned;
            assigned = value;
            value[0] += 9;
            value[1] -= 7;
            assert(original == assigned && value != original);
            vect2& self = assigned;
            assigned = self;
            assert(assigned == original);
            const vect2 other(y, -x);
            assert(original + other == vect2(x + y, y - x));
            assert(original - other == vect2(x - y, y + x));
            assert(-original == vect2(-x, -y));
            for (int factor = -2; factor <= 2; ++factor) {
                assert(original * factor == vect2(x * factor, y * factor));
                assert(factor * original == original * factor);
                value = original;
                assert(&(value *= factor) == &value);
                assert(value == original * factor);
            }
            value = original;
            assert(&(value += other) == &value);
            assert(value == original + other);
            assert(&(value -= other) == &value && value == original);
            assert(value++ == original && value == vect2(x + 1, y + 1));
            assert(&++value == &value && value == vect2(x + 2, y + 2));
            assert(value-- == vect2(x + 2, y + 2));
            assert(&--value == &value && value == original);
            value += value;
            assert(value == original * 2);
            value -= value;
            assert(value == vect2());
            std::ostringstream out;
            assert(&(out << original) == &out);
            assert(out.str() == "{" + std::to_string(x) + ", " + std::to_string(y) + "}");
        }
    }
}
