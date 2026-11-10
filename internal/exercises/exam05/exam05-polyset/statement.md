# polyset

Extend the supplied array-backed and tree-backed bags with membership queries,
then build a non-owning set wrapper. The supplied interfaces, implementations,
and `main.cpp` were written for Codedrill; they are not school-provided files.

Write `searchable_array_bag.{hpp,cpp}` and `searchable_tree_bag.{hpp,cpp}`.
Each class publicly inherits its corresponding concrete bag and `searchable_bag`.
Implement `bool has(int) const`. Use orthodox canonical form: declare and define
default construction, copy construction, copy assignment, and destruction.
Copies must own independent storage; self-assignment must preserve contents.

Write `set.{hpp,cpp}`. Construct it with a `searchable_bag&`. Offer `insert(int)`,
`insert(int*, int)`, `has(int) const`, `print() const`, `clear()`, and
`const searchable_bag& get_bag() const`. Each insertion skips values already
present, including duplicates within one bulk insertion. Forward the remaining
operations to the original bag. Do not delete that bag when the wrapper dies.
The subject requests orthodox canonical form for all classes. Its provided main
does not establish default/copy semantics for the reference-holding `set`.
The reference deletes those three operations; the tests do not require that
choice or add a prohibition on copying to the subject.

Leave the supplied files unchanged. Array printing retains insertion order;
tree printing is sorted. Both print each value followed by a space, then a
newline. The array bag retains duplicates. The supplied tree implementation
discards duplicates, omits zero when printing (but can find it), and writes
`create node: N`, `duplicate value: delete node`, and `destroying value: N`
messages. These behaviors reproduce the local subject resources; do not add
those messages to your searchable classes or set. `extract_tree()` transfers
the tree root out; `set_tree()` destroys the previous tree and takes that root.
The supplied base classes do not have virtual destructors: destroy concrete
objects rather than deleting through `bag*` or `searchable_bag*`.

Tests check behavior, const access, copying populated bags, assignment, clearing,
and destruction. Review canonical-form declarations separately. Use the actual
exam's supplied interfaces and requirements when preparing a school submission.
