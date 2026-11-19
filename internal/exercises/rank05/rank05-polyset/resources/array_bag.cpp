#include "array_bag.hpp"
#include <iostream>
#include <memory>
#include <utility>

array_bag::array_bag() : data(nullptr), size(0) {}
array_bag::array_bag(const array_bag& source) : data(nullptr), size(0) {
    insert(source.data, source.size);
}
array_bag& array_bag::operator=(const array_bag& source) {
    if (this != &source) {
        array_bag replacement(source);
        std::swap(data, replacement.data);
        std::swap(size, replacement.size);
    }
    return *this;
}
array_bag::~array_bag() { delete[] data; }
void array_bag::insert(int value) { insert(&value, 1); }
void array_bag::insert(int* values, int count) {
    if (count <= 0) return;
    std::unique_ptr<int[]> replacement(new int[size + count]);
    for (int index = 0; index < size; ++index) replacement[index] = data[index];
    for (int index = 0; index < count; ++index) replacement[size + index] = values[index];
    delete[] data;
    data = replacement.release();
    size += count;
}
void array_bag::print() const {
    for (int index = 0; index < size; ++index) std::cout << data[index] << ' ';
    std::cout << '\n';
}
void array_bag::clear() { delete[] data; data = nullptr; size = 0; }
