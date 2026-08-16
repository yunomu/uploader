module View.Index exposing (view)

import Element exposing (Element)


view : Element msg
view =
    Element.column []
        [ Element.link []
            { url = "/files"
            , label = Element.text "Files"
            }
        ]
