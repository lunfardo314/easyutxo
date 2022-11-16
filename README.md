# EasyUTXO
A simple yet extendable, programmable and powerful UTXO ledger model with advanced concepts

_EasyUTXO_ at this stage is PoC, however it has intention to become a fully-functional UTXO ledger used in DLT projects.

_EasyUTXO_ uses [EasyFL](https://github.com/lunfardo314/easyfl) as the language for its functional constraints. Here is 
a [dexription of EasyFL](https://hackmd.io/@Evaldas/S14WHOKMi). 

## Introduction

The _EasyUTXO_ is an implementation of the UTXO concept well-known in distributed ledger technologies, such as Bitcoin or 
IOTA (Stardust release). 

The _EasyUTXO_ model of the UTXO ledger intends to go beyond the Stardust ledger model, 
which is hardcoded and language-dependent to certain extent. The _EasyUTXO_ introduces **programmability** of the ledger transition 
function (transaction model), while same time preserving robustness and determinism of the UTXO ledger model 
by adding **formal verifiability** and **minimizing the global trust assumptions**. 
The reference code _EasyUTXO_ model is written in Go, however the transaction model (UTXO types/behavior) are completely 
platform independent.

The above is achieved by:
* building very generic and minimalistic transaction model (the minimization of assumptions)
* programming almost all the state transition logic as the _finite functional constraints_, expressed in the
[EasyFL](https://github.com/lunfardo314/easyfl) functional constraint language. 

The whole model of the state transition is finite and non-Turing complete, because EasyFL language does not allow loops and recursions.

This makes _EasyFL_ program (expression) and its specification is the same thing. With this trait of the *EasyUTXO* makes 
it possible **automatic prove** of the ledger properties, e.g. if it can achieve desired (or undesired) states.
For example the answer to the questions _Is it possible to spend funds without knowing private key?_ 
or _Is the output with specific constraint unlockable?_ can be automated. This is not possible with Turing-complete models
without adding formal specifications to the program code./

This way essentially all the Stardust UTXO types are expressed in provable _EasyFL_ constraints: 
all kind of transfers, chain constraints, aliases, NFTs, time lock, expiry conditions etc 
(by the time of writing, native tokens and foundries remains in future plans).

This also makes extension and programmability of the UTXO transactions by embedding new constraints right into the transaction (inline),
or extending library of constrains with new constraints, available globally.

The functional nature of the _EasyFL_ makes it easy to **extend the computation model** of the ledger even further, for example
with Turing-complete computations of _WebAssembly_, _EVM_, _Move_ or any other. 

The above makes _EasyUTXO_ a powerful yet safe architectural abstraction layer for any further developments of the UTXO ledgers.

## Transaction model

### Output
UTXO ledger consists of UTXOs (**U**nspent **T**ransa**X**ion **O**utputs). Each output is an array of _constraints_:
_(C1, C1m ..., CN)_. Each constraint _Ci_ is an _EasyFL_ expression (formula) in its bytecode form.

For the output to be valid, all constraints of the output are evaluated. If all formulas does not panic and return _true_, 
the output is valid. If constraint evaluates to _false_ or panics, the whole output is invalidated.

Each constraint is evaluated by supplying to the *EasyFL* engine:
* global context of the transaction being validated
* local place (path) in the transaction where being evaluated constraint is located. This constraints the data in the local context. 
For example, constraint `addressED25519(0x12345...)` by knowing its location, reaches out the corresponding _unlock block_ or signature
to check if the signature is valid

Any data which appears in the output comes in the form of constraint, i.e. is runnable. Examples of constraints:
* `amount(1000)` check validity of the amount
* `timestamp(123456)` checks validity of the timestamp in the output
* `chainLock(0xaaaaaaaaaaaaaa)` check if output can be consumed, i.e. if the chain `0xaaaaaaaaaaaaa..` is transited in the same transaction

### Transaction
Ledger is updated in atomic units, called _transaction_. Each transaction consist of:
* _consumed outputs_, up to 255
* _produced outputs_, up to 255
* _transaction level data_, such as transaction timestamp, _input commitment_, user signature, provided constraint scripts etc

We distinguish full _validation context of the transaction_ and _transaction_ it self. 

_Validation context_ includes all consumed inputs in its entirety. It is necessary for validation of the transaction.
The _transaction_ itself only contains _inputIDs_ instead of consumed outputs. The _transaction_ is the data transferred over the wire
between nodes. To validate it, the node rebuild _validation context_ on its own copy of the ledger state.

### Validation of the transaction

The node runs the following steps fo each transaction received by the wire:
* rebuilds _validation context_
* it evaluates all constraints on each consumed output 
* it evaluates all constraints on each produced output 
* checks integrity of global constraints, which cannot be expressed in EasyFL (unbounded):
  * balances of `amount(N)` constraints must be equal on consumed and produced side
  * each outputs must have one of predefined locks (addresses) 
  * _checks validity of the input commitment_

The above loop is the only global assumption of the ledger. It is completely agnostic of what constraint are provided. 
This way UTXO behavior is encoded into the constraints on outputs. 

### Examples

Example of user-readable transaction printout:
```
  Transaction. ID: 32xdf79cf3a04c4030d733553db328611413929da07c0be6ee5ac6a3385f0d45c42, size: 390
  Timestamp: 4x636b89b1 (1667991985)
  Input commitment: 32x8152eb3defc209036763c044e2b509953708536458a39e07434dc397bc800bb8
  Inputs (consumed outputs): 
    #0: [1]32xf0a31a9cf03579b51e5365b5738a8c7980c7f2e8bebcdc62766a6e31d1bffb84 (97 bytes)
       0: amount(u64/2000) (11 bytes)
       1: timestamp(u32/1667991981) (7 bytes)
       2: addressED25519(0xb400219a67085d3a22c1fee43844ab06995e1e1d2384b3f98ceddba6e2b75273) (35 bytes)
       3: chain(0x0000000000000000000000000000000000000000000000000000000000000000ffffff) (38 bytes)
       Unlock data: [0x,0x,1xff,3x000300]
    #1: [1]32xc3b6521c9415f8be70d15de2827d66b46e9167b6ecec9ed4e4b60e64774363e2 (58 bytes)
       0: amount(u64/1000) (11 bytes)
       1: timestamp(u32/1667991984) (7 bytes)
       2: chainLock(0xb3051a25fef3ee5163c469646600323e654b871e8aec34330ce85e81ec42fc04) (35 bytes)
       Unlock data: [0x,0x,2x0003]
  Outputs (produced): 
    #0 (97 bytes) :
       0: amount(u64/2500) (11 bytes)
       1: timestamp(u32/1667991985) (7 bytes)
       2: addressED25519(0xb400219a67085d3a22c1fee43844ab06995e1e1d2384b3f98ceddba6e2b75273) (35 bytes)
       3: chain(0xb3051a25fef3ee5163c469646600323e654b871e8aec34330ce85e81ec42fc04000300) (38 bytes)
    #1 (58 bytes) :
       0: amount(u64/500) (11 bytes)
       1: timestamp(u32/1667991985) (7 bytes)
       2: addressED25519(0xb400219a67085d3a22c1fee43844ab06995e1e1d2384b3f98ceddba6e2b75273) (35 bytes)
```

Example of output constraint in _EasyFL_ (`timelock` in this case):
```go
// enforces output can be unlocked only after specified time
// $0 is Unix seconds of the time lock
func timelock: or(
    and(
        selfIsProducedOutput,
        equal(len8($0), 4),             // must be 4-bytes long
        lessThan(txTimestampBytes, $0)  // time lock must be after the transaction (not very necessary)
    ),
    and(
        selfIsConsumedOutput,
        lessThan($0, txTimestampBytes)  // is unlocked if tx timestamp is strongly after the time lock 
    )
)
```