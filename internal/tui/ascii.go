// Dragon ASCII art for atelier Mission Control welcome screen. User-provided.
// Source: .atl/dragon-art.txt (42 lines x 100 cols, 7-bit ASCII, LF endings).
//
// This file is BRAND-ONLY. The dragon is identity, not a UI component.
// It appears once on the welcome screen and never in interactive widgets.
// Brand color constants live in styles.go under "// --- Dragon Brand Accents ---".
//
// Branding character extends beyond UI palette. Dragon accents are explicitly
// separate from canonical Catppuccin Mocha to prevent contamination of UI styling.
package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// DragonArt is the welcome-screen dragon silhouette.
// Canvas: 42 lines x 100 columns. 7-bit ASCII only (no backticks).
// See design §13.1 for the dimensional contract.
// Contents read verbatim from .atl/dragon-art.txt at apply time.
const DragonArt = `                                                                       :                            
                                                                     :+                             
                                                                    #-                              
                                                                  %%          -*                    
                                                               =@%         :#%                      
                                                            -%@#: =      +@%:                       
                                                         -%@@++%#     +@@%:                         
                                                     :#@@@@@@#    :*@@%-                            
                                                 =#@@@%+-     -#@@@#: :-=                           
                                             -#@@@%=     =#@@@@@@@%@#=                              
                                           #=      -#@@@@@@@@@@*:                                   
                                          :-+%@@@@@@%%#+-                                           
                                      +@@@@@@#+-       -+ %@@@@%*=:                                 
                                   +@@@@%- :=*###%%%@@@# :@@@@@@@@@@@#=-                            
                                 *@@@% #@@@@@@@@@@@@@@:                                             
                               :@@@@%@@@@@@@@@@@@@%=  --  :#@@@*-                                   
                              +@@@@@@@@@@@%#*+-:   +@@@@@%-  +@@@@@+                                
                             *@@@@@#- :=+=:     :=%@@@@@@@@@=  =@@@@@#                              
                            +@@@#:#@@@@@@@@@@%+ +#:  :#@@@@@@@:  #@@@@@#                            
                           :@@# =@@@@@@*       - =@@%-  -%@@@@@#  -@@@@@@-                          
                          :@@=  @@@@@=   -@@@@*   +@@@@+  :%@@@@%   @@@@@@#                         
                         :@@- :@@@@@-   #@@@@@@@:  @- @@@=  -@@@@@   @@-%@@@                        
                        *@@%%@@@@@@#   +@@@@@@@@=  *   =@@%   %@@@@  -@@%  :##                      
                     -#@@@@@@@@@@@@+   *@@@@@@@@-       :@@@:  #@@@@  +@@@:                         
                 @@@@@@@@@@@@@@+:       #@@@@@@%          @@@   %@@@=  @@@%                         
                -@@@-%@@@@@@@@%          :@@@@#           =@@%  :@@@@  =@@@*                        
                -@@@@@@@@@@# +%@#:   :=%@@@@#              %@@*  +@@@- :@@@@:                       
                :@@@@@@@## :     =%@@@@@%+:                #@@@  :@@@+  @@@@#                       
                 %@%=@+:-        *@@*:                     *@#@-  @@@*  @@@@%                       
                              :@%:                         ** @+  %@@*  @@+#@#                      
                            :#+                            +- @#  @@@+ :@@@  --                     
                           :=                              :  @*  @@@= +@@@:                        
                                                             :@- -@@@  @@@@                         
                                                             #@: *@@+ +@@@#                         
                                                            -@+ -@@@ -@@@@-                         
                                                            %%  %@@  @@=@%                          
                                                           #%  #@@  @@# @*                          
                                                          ##  %@% -@@@:  +                          
                                                         %= :@@+ =@@@-                              
                                                       =*  *@#   @@@:                               
                                                     :-  +@+    =@+                                 
                                                      :+=       *-                                  `

// DragonRows is the line count of DragonArt, computed at init from the const.
// Used by view.go to decide whether the terminal is large enough for the full welcome.
var DragonRows = strings.Count(DragonArt, "\n") + 1

// DragonCols is the width of the first line of DragonArt, computed at init.
// All lines are uniform width; using the first line is sufficient.
// Used by view.go to decide whether the terminal is wide enough for the full welcome.
var DragonCols = func() int {
	if i := strings.Index(DragonArt, "\n"); i >= 0 {
		return utf8.RuneCountInString(DragonArt[:i])
	}
	return utf8.RuneCountInString(DragonArt)
}()

// RenderDragon applies the given lipgloss style to DragonArt and returns the colored string.
// The caller decides where to place it (typically via lipgloss.Place at center).
func RenderDragon(style lipgloss.Style) string {
	return style.Render(DragonArt)
}
