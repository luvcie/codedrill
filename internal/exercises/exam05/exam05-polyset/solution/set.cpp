#include "set.hpp"

set::set(searchable_bag& backing) : storage(backing) {}
void set::insert(int value) { if (!has(value)) storage.insert(value); }
void set::insert(int* values, int count) {
    for (int index = 0; index < count; ++index) insert(values[index]);
}
bool set::has(int value) const { return storage.has(value); }
void set::print() const { storage.print(); }
void set::clear() { storage.clear(); }
const searchable_bag& set::get_bag() const { return storage; }
