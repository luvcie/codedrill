#include "bigint.hpp"
#include <cassert>
#include <sstream>
#include <string>

static std::string text(const bigint& value) {
    std::ostringstream out;
    assert(&(out << value) == &out);
    return out.str();
}

int main() {
    assert(text(bigint()) == "0");
    for (unsigned a = 0; a < 200; a += 7) {
        for (unsigned b = 0; b < 200; b += 11) {
            const bigint left(a), right(b);
            assert(text(left + right) == std::to_string(a + b));
            assert((left == right) == (a == b));
            assert((left != right) == (a != b));
            assert((left < right) == (a < b));
            assert((left > right) == (a > b));
            assert((left <= right) == (a <= b));
            assert((left >= right) == (a >= b));
            bigint value(left), copy;
            copy = value;
            assert(&(value += right) == &value && value == left + right);
            assert(copy == left);
            assert(copy++ == left && copy == left + 1);
            assert(&++copy == &copy && copy == left + 2);
            bigint& self = value;
            value = self;
            value += value;
            assert(value == bigint((a + b) * 2));
        }
    }
    const bigint power = bigint(1) << 90;
    const bigint nines = (bigint(9) << 89) + (bigint(9) << 88);
    assert(text(power) == "1" + std::string(90, '0'));
    assert(text((power + nines) >> 88) == "199");
    bigint carry(999999999);
    ++carry;
    assert(text(carry) == "1000000000");
    bigint shifted(731);
    assert(&(shifted <<= bigint(3)) == &shifted && text(shifted) == "731000");
    assert(&(shifted >>= bigint(4)) == &shifted && text(shifted) == "73");
    assert((shifted >> power) == bigint());
    assert((bigint() << power) == bigint());
    assert((shifted >> 0) == shifted && (shifted << 0) == shifted);
    assert(bigint(73) == bigint(73) && !(bigint(73) > bigint(73)));
}
