#include "bigint.hpp"
#include <limits>
#include <stdexcept>

bigint::bigint(unsigned int value) : decimal(std::to_string(value)) {}

bigint& bigint::operator+=(const bigint& rhs) {
    const std::size_t width = decimal.size() > rhs.decimal.size() ? decimal.size() : rhs.decimal.size();
    std::string result(width + 1, '0');
    unsigned carry = 0;
    for (std::size_t offset = 0; offset < width; ++offset) {
        unsigned column = carry;
        if (offset < decimal.size()) column += decimal[decimal.size() - offset - 1] - '0';
        if (offset < rhs.decimal.size()) column += rhs.decimal[rhs.decimal.size() - offset - 1] - '0';
        result[width - offset] = '0' + column % 10;
        carry = column / 10;
    }
    result[0] = '0' + carry;
    decimal = carry ? result : result.substr(1);
    return *this;
}
bigint bigint::operator+(const bigint& rhs) const { return bigint(*this) += rhs; }
bigint& bigint::operator++() { return *this += 1; }
bigint bigint::operator++(int) { bigint before(*this); ++*this; return before; }

std::size_t bigint::shift_count(const bigint& count) const {
    std::size_t result = 0;
    const std::size_t limit = std::numeric_limits<std::size_t>::max();
    for (char digit : count.decimal) {
        const unsigned value = digit - '0';
        if (result > (limit - value) / 10) return limit;
        result = result * 10 + value;
    }
    return result;
}
bigint& bigint::operator<<=(const bigint& count) {
    if (decimal == "0") return *this;
    const std::size_t amount = shift_count(count);
    if (amount > decimal.max_size() - decimal.size()) throw std::length_error("decimal shift");
    decimal.append(amount, '0');
    return *this;
}
bigint& bigint::operator>>=(const bigint& count) {
    const std::size_t amount = shift_count(count);
    if (amount >= decimal.size()) decimal = "0";
    else decimal.resize(decimal.size() - amount);
    return *this;
}
bigint bigint::operator<<(const bigint& count) const { return bigint(*this) <<= count; }
bigint bigint::operator>>(const bigint& count) const { return bigint(*this) >>= count; }
bool bigint::operator==(const bigint& rhs) const { return decimal == rhs.decimal; }
bool bigint::operator!=(const bigint& rhs) const { return !(*this == rhs); }
bool bigint::operator<(const bigint& rhs) const {
    return decimal.size() == rhs.decimal.size() ? decimal < rhs.decimal : decimal.size() < rhs.decimal.size();
}
bool bigint::operator>(const bigint& rhs) const { return rhs < *this; }
bool bigint::operator<=(const bigint& rhs) const { return !(rhs < *this); }
bool bigint::operator>=(const bigint& rhs) const { return !(*this < rhs); }
std::ostream& operator<<(std::ostream& out, const bigint& value) { return out << value.decimal; }
