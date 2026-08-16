module View.Atom.Button exposing (button, updateButton)

import Element exposing (Attribute, Element)
import Element.Background
import Element.Border
import Element.Font
import Element.Input


buttonBase : List (Attribute msg) -> msg -> String -> Element msg
buttonBase attrs msg label =
    Element.Input.button
        (List.append attrs
            [ Element.Border.rounded 4
            , Element.Border.width 1
            , Element.paddingXY 5 1
            ]
        )
        { onPress = Just msg
        , label = Element.text label
        }


button : msg -> String -> Element msg
button =
    buttonBase []


updateButton : msg -> String -> Element msg
updateButton =
    buttonBase
        [ Element.Font.color (Element.rgb 1 1 1)
        , Element.Background.color (Element.rgb 0 0 1)
        ]
