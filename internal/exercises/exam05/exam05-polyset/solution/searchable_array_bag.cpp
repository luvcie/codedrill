#include "searchable_array_bag.hpp"

searchable_array_bag::searchable_array_bag() = default;
searchable_array_bag::searchable_array_bag(const searchable_array_bag& source) : array_bag(source) {}
searchable_array_bag& searchable_array_bag::operator=(const searchable_array_bag& source) {
    array_bag::operator=(source);
    return *this;
}
searchable_array_bag::~searchable_array_bag() = default;
bool searchable_array_bag::has(int value) const {
    for (int index = size; index > 0; --index) {
        if (data[index - 1] == value) return true;
    }
    return false;
}
