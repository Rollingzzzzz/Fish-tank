// G6.2 (partial, G3.5 lane B): package config — owns config.json beside the
// exe, the ZCode key import (zcode.go) and (in a later goal) the single-
// instance lock. Lane B only delivers the key import; config.go loading and
// the lock are added by G6.2.
package config
