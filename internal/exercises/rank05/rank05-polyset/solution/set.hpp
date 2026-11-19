#pragma once
#include "searchable_bag.hpp"

class set {
    searchable_bag& storage;
public:
    set(searchable_bag& backing);
    set() = delete;
    set(const set&) = delete;
    set& operator=(const set&) = delete;
    ~set() = default;
    void insert(int value);
    void insert(int* values, int count);
    bool has(int value) const;
    void print() const;
    void clear();
    const searchable_bag& get_bag() const;
};
