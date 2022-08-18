# EasyUTXO
A simple UTXO ledger with  advanced concepts

# Drafts

Transaction structure:
Data element DE 

Data element ::= block | array of Data element 
* Transaction
  * Essence ::= data element
    * Inputs := array of Input
    * Outputs := array of Output
  * UnlockBlocks := array of UnlockBlock
  * Other data elements

Inputs 
Output is array of data elements:
* Blocks := array of Block
* Assets