module View.Atom.Table exposing (view)

import Element exposing (Element)
import Element.Border as Border


columnHeight : Element.Length
columnHeight =
    Element.px 30


view :
    List record
    -> (Int -> record -> Bool)
    ->
        List
            { header : String
            , view : Int -> record -> Element msg
            }
    -> Element msg
view data emphasis columns =
    Element.indexedTable [ Element.spacingXY 0 5 ]
        { data = data
        , columns =
            List.map
                (\c ->
                    { header =
                        Element.el
                            [ Element.paddingXY 10 2
                            , Element.height columnHeight
                            , Border.widthEach
                                { bottom = 2
                                , top = 2
                                , left = 0
                                , right = 0
                                }
                            ]
                        <|
                            Element.text c.header
                    , width = Element.shrink
                    , view =
                        \i r ->
                            Element.el
                                [ Element.paddingXY 10 2
                                , Element.height columnHeight
                                , Border.solid
                                , Border.widthEach
                                    { bottom =
                                        if emphasis i r then
                                            3

                                        else
                                            1
                                    , left = 0
                                    , right = 0
                                    , top = 0
                                    }
                                ]
                            <|
                                c.view i r
                    }
                )
                columns
        }
