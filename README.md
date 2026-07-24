ignore file for the codehound 
- Line: // codehound-ignore: RULE_ID on same line or line before
- File: // codehound-ignore-file or // codehound-ignore-file: RULE_ID in header (first 20 lines)
- Block: // codehound-ignore-start: RULE_ID ... // codehound-ignore-end
Replace RULE_ID with a rule like BP-1 or all to suppress everything. Go uses //, Python uses #.