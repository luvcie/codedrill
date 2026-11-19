#include "tree_bag.hpp"
#include <iostream>
#include <utility>

tree_bag::tree_bag() : tree(nullptr) {}
tree_bag::tree_bag(const tree_bag& source) : tree(clone(source.tree)) {}
tree_bag& tree_bag::operator=(const tree_bag& source) {
    if (this != &source) {
        tree_bag replacement(source);
        std::swap(tree, replacement.tree);
    }
    return *this;
}
tree_bag::~tree_bag() { destroy(tree); }
tree_bag::node* tree_bag::extract_tree() {
    node* result = tree;
    tree = nullptr;
    return result;
}
void tree_bag::set_tree(node* replacement) {
    destroy(tree);
    tree = replacement;
}
tree_bag::node* tree_bag::clone(const node* root) {
    if (!root) return nullptr;
    node* result = new node{nullptr, nullptr, root->value};
    try {
        result->l = clone(root->l);
        result->r = clone(root->r);
    } catch (...) {
        destroy(result);
        throw;
    }
    return result;
}
void tree_bag::destroy(node* root) {
    if (!root) return;
    std::cout << "destroying value: " << root->value << '\n';
    destroy(root->l);
    destroy(root->r);
    delete root;
}
void tree_bag::insert(int value) {
    std::cout << "create node: " << value << '\n';
    node** position = &tree;
    while (*position) {
        if ((*position)->value == value) {
            std::cout << "duplicate value: delete node\n";
            return;
        }
        position = value < (*position)->value ? &(*position)->l : &(*position)->r;
    }
    *position = new node{nullptr, nullptr, value};
}
void tree_bag::insert(int* values, int count) {
    for (int index = 0; index < count; ++index) insert(values[index]);
}
void tree_bag::print_branch(const node* root) {
    if (!root) return;
    print_branch(root->l);
    if (root->value != 0) std::cout << root->value << ' ';
    print_branch(root->r);
}
void tree_bag::print() const { print_branch(tree); std::cout << '\n'; }
void tree_bag::clear() { destroy(tree); tree = nullptr; }
