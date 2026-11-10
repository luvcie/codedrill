#include "vect2.hpp"

vect2::vect2(int x, int y) : coordinates{x, y} {}
int& vect2::operator[](int index) { return coordinates[index]; }
int vect2::operator[](int index) const { return coordinates[index]; }

vect2& vect2::operator+=(const vect2& rhs) {
    coordinates[0] += rhs[0];
    coordinates[1] += rhs[1];
    return *this;
}
vect2& vect2::operator-=(const vect2& rhs) {
    coordinates[0] -= rhs[0];
    coordinates[1] -= rhs[1];
    return *this;
}
vect2& vect2::operator*=(int factor) {
    coordinates[0] *= factor;
    coordinates[1] *= factor;
    return *this;
}
vect2 vect2::operator+(const vect2& rhs) const { return vect2(*this) += rhs; }
vect2 vect2::operator-(const vect2& rhs) const { return vect2(*this) -= rhs; }
vect2 vect2::operator-() const { return vect2(-coordinates[0], -coordinates[1]); }
vect2 vect2::operator*(int factor) const { return vect2(*this) *= factor; }
vect2& vect2::operator++() { return *this += vect2(1, 1); }
vect2 vect2::operator++(int) { vect2 old = *this; ++*this; return old; }
vect2& vect2::operator--() { return *this -= vect2(1, 1); }
vect2 vect2::operator--(int) { vect2 old = *this; --*this; return old; }
bool vect2::operator==(const vect2& rhs) const {
    return coordinates[0] == rhs[0] && coordinates[1] == rhs[1];
}
bool vect2::operator!=(const vect2& rhs) const { return !(*this == rhs); }
vect2 operator*(int factor, const vect2& value) { return value * factor; }
std::ostream& operator<<(std::ostream& out, const vect2& value) {
    return out << '{' << value[0] << ", " << value[1] << '}';
}
