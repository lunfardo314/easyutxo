# Sketch of Easy UTXO Script (EUS)

## Purpose
_EUS_ is a non-Turing-equivalent processor (VM) with instructions and scripting available in binary and symbolical assembly form.
`Non-Turing-equivalent` mean that it's execution is always capped even in the unbounded environment space/time-wise.
This means formal verification and gas-budget assessment is possible.

## Validation context
EUS scripts are executed with the purpose to validate special tree-like read-only structures, `ValidationContext`.
Validation context is a representation of the data structure `LazyTree`.  

The `LazyTree` is a tree where each node has no more than 256 children. Each node may be interpreted as raw byte array. 
The trees with children are also interpreted as a serialized form (byte array) of `LazyArray`, 
where each child refers to the element of the array.

This way each node from the root can be reached along the path `L = [b1, b2, ..., bn]` where each `bi` is a byte value from 0 to 255.
`L` is the location of the element. The location of the element as valid as long as it is possible parse all 
raw bytes (serialized) of nodes in the path to correct `LazyArray`. The leaf of the tree are interpreted as just raw byte arrays.

This way `LazyTree` represents a static, bounded data structure where each element can be addresses by 

## Validation
_EUS_ script is a sequence of instructions, executed one by one from left to right. The instruction set does not allow loops.
Each EUS script has a deterministic binary (serialized) form, a byte array.

The `ValidationContext` is validated by scripts contained:
* as elements of the `LazyTree` of itself
* as _well known code from the library_ referenced by short reference in the code. 

During execution, each EUS script has access to the whole `ValidationContext` it is validating. 

The outcome of execution is either `ok` or `fail`. For the `ValidationContext` to be valid, all scripts must produce `ok`.

Each `ValidationContext` is use case specific. It is a hadcoded part of the VM just like the engine:
* it defines the order how the script contained in it are run
* it imposes global validation constrains which cannot be expressed by scripts
* it contains the library of `well-knonw-codes`

## Engine structure

* Stack:
  * stack element is byte slice/array, may be empty. 
  * Max element length is MaxDataLen = 2^14-1
  * stack depth is limited MaxStackDepth
* Registers
  * number of registers is 256
  * register value is byte array/slice, may be empty
* Initially, upon script start:
  * stack is empty
  * register 0 contains location of the script in the validation environment
  * all other registries contain empty values
