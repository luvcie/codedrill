#ifndef CODEDRILL_BIGINT_HPP
#define CODEDRILL_BIGINT_HPP
#include <iostream>
#include <string>

class bigint {
    std::string decimal;
    std::size_t shift_count(const bigint& count) const;
public:
    bigint(unsigned int value = 0);
    bigint operator+(const bigint& rhs) const;
    bigint& operator+=(const bigint& rhs);
    bigint& operator++();
    bigint operator++(int);
    bigint operator<<(const bigint& count) const;
    bigint operator>>(const bigint& count) const;
    bigint& operator<<=(const bigint& count);
    bigint& operator>>=(const bigint& count);
    bool operator==(const bigint& rhs) const;
    bool operator!=(const bigint& rhs) const;
    bool operator<(const bigint& rhs) const;
    bool operator>(const bigint& rhs) const;
    bool operator<=(const bigint& rhs) const;
    bool operator>=(const bigint& rhs) const;
    friend std::ostream& operator<<(std::ostream& out, const bigint& value);
};
#endif
