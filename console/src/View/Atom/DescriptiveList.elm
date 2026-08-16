module View.Atom.DescriptiveList exposing (view)

import Element exposing (Element)
import Element.Font


edge =
    { top = 0, bottom = 0, right = 0, left = 0 }


view :
    List
        { header : String
        , body : Element msg
        }
    -> Element msg
view =
    Element.column [ Element.spacing 20 ]
        << List.map
            (\a ->
                Element.column [ Element.spacing 10 ]
                    [ Element.el [ Element.Font.bold ] <| Element.text <| a.header ++ ":"
                    , Element.el [ Element.paddingEach { edge | left = 30 } ] a.body
                    ]
            )
