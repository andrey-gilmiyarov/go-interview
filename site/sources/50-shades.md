# Карта 66 тематических якорей 50 Shades of Go

В первом столбце ссылка ведёт к исходному H6-разделу; его точный якорь остаётся в URL. Русские подписи кратко описывают тему и не копируют исходное оглавление.

«Включено» означает, что существующая статья справочника уже разбирает тот же концепт; остальные пункты — дополнительное чтение. Возраст источника сам по себе не делает раздел историческим. Оговорки отмечают контекст старых или платформенных примеров.

| № и ссылка на раздел | Краткая русская тема | Связанная тема справочника | Статус и версия |
| --- | --- | --- | --- |
| [1](https://golang50shades.com/#anameopening_bracesaopeningbracecantbeplacedonaseparateline) | Перенос открывающей скобки | [Сборка и toolchain](/go/tooling/toolchain-build) | Дополнительное чтение |
| [2](https://golang50shades.com/#anameunused_varsaunusedvariables) | Неиспользованное имя | [Сборка и toolchain](/go/tooling/toolchain-build) | Дополнительное чтение |
| [3](https://golang50shades.com/#anameunused_importsaunusedimports) | Импорт без применения | [Сборка и toolchain](/go/tooling/toolchain-build) | Дополнительное чтение |
| [4](https://golang50shades.com/#anameshort_varsashortvariabledeclarationscanbeusedonlyinsidefunctions) | Краткое объявление внутри функции | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [5](https://golang50shades.com/#anamevars_redeclarearedeclaringvariablesusingshortvariabledeclarations) | Повторное объявление в блоке | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [6](https://golang50shades.com/#anameshort_fieldsacantuseshortvariabledeclarationstosetfieldvalues) | Присваивание полю структуры | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [7](https://golang50shades.com/#anamevars_shadowaaccidentalvariableshadowing) | Случайная тень внешнего имени | [Циклы и range](/go/language/control-range) | Дополнительное чтение |
| [8](https://golang50shades.com/#anamenil_initacantuseniltoinitializeavariablewithoutanexplicittype) | Тип для нулевого значения | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [9](https://golang50shades.com/#anamenil_slices_mapsausingnilslicesandmaps) | Нулевые срезы и карты | [Срезы](/go/language/slices), [карты](/go/language/maps) | Включено; Обе коллекции могут быть nil; запись в nil-map и append в nil-срез различаются. |
| [10](https://golang50shades.com/#anamemap_capamapcapacity) | Подсказка ёмкости map | [Карты](/go/language/maps) | Дополнительное чтение |
| [11](https://golang50shades.com/#anamenil_stringsastringscantbenil) | Строки не бывают nil | [Строки и руны](/go/language/strings-runes) | Дополнительное чтение |
| [12](https://golang50shades.com/#anamearray_func_argsaarrayfunctionarguments) | Копирование массива в аргументе | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [13](https://golang50shades.com/#anameunexpected_slice_arr_valsaunexpectedvaluesinsliceandarrayrangeclauses) | Значения, выдаваемые range | [Циклы и range](/go/language/control-range) | Включено |
| [14](https://golang50shades.com/#anameone_dim_slice_arraslicesandarraysareonedimensional) | Массивы и срезы одномерны | [Срезы](/go/language/slices) | Дополнительное чтение |
| [15](https://golang50shades.com/#anamemap_key_neaaccessingnonexistingmapkeys) | Проверка отсутствующего ключа | [Карты](/go/language/maps) | Дополнительное чтение |
| [16](https://golang50shades.com/#anameimm_stringsastringsareimmutable) | Неизменяемость строки | [Строки и руны](/go/language/strings-runes) | Дополнительное чтение |
| [17](https://golang50shades.com/#anamestring_byte_slice_convaconversionsbetweenstringsandbyteslices) | Обмен между строкой и []byte | [Строки и руны](/go/language/strings-runes) | Дополнительное чтение |
| [18](https://golang50shades.com/#anamestring_idxastringsandindexoperator) | Индекс строки читает байт | [Строки и руны](/go/language/strings-runes) | Дополнительное чтение |
| [19](https://golang50shades.com/#anamestrings_na_utfastringsarenotalwaysutf8text) | Валидность UTF-8 | [Строки и руны](/go/language/strings-runes) | Включено |
| [20](https://golang50shades.com/#anamestring_lengthastringlength) | Длина в байтах и кодовых точках | [Строки и руны](/go/language/strings-runes) | Включено |
| [21](https://golang50shades.com/#anamemline_lit_commaamissingcommainmultilineslicearrayandmapliterals) | Запятая в многострочном литерале | [Сборка и toolchain](/go/tooling/toolchain-build) | Дополнительное чтение |
| [22](https://golang50shades.com/#anamelog_fatal_exitalogfatalandlogpanicdomorethanlog) | Завершение процесса при fatal-логе | [Структурированные логи](/go/stdlib/slog) | Дополнительное чтение |
| [23](https://golang50shades.com/#anamecoll_no_syncabuiltindatastructureoperationsarenotsynchronized) | Встроенные коллекции и синхронизация | [Синхронизация](/go/concurrency/sync-primitives) | Дополнительное чтение |
| [24](https://golang50shades.com/#anamestring_range_valsaiterationvaluesforstringsinrangeclauses) | Смещение и руна при обходе строки | [Строки и руны](/go/language/strings-runes) | Включено |
| [25](https://golang50shades.com/#anamemap_rangeaiteratingthroughamapusingaforrangeclause) | Неустановленный порядок карты | [Карты](/go/language/maps) | Включено |
| [26](https://golang50shades.com/#anameswitch_fallafallthroughbehaviorinswitchstatements) | Переход switch к следующей ветви | [Циклы и range](/go/language/control-range) | Дополнительное чтение |
| [27](https://golang50shades.com/#anameinc_decaincrementsanddecrements) | Постфиксная запись инкремента | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [28](https://golang50shades.com/#anamebit_notabitwisenotoperator) | Побитовое дополнение числа | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [29](https://golang50shades.com/#anameop_precedenceaoperatorprecedencedifferences) | Приоритет операторов | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [30](https://golang50shades.com/#anameunexp_struct_field_encaunexportedstructurefieldsarenotencoded) | Экспорт полей при JSON-кодировании | [JSON](/go/stdlib/json) | Дополнительное чтение; Формулировка о полях относится к `encoding/json` v1; поведение v2 сверяйте отдельно. |
| [31](https://golang50shades.com/#anamegor_app_exitaappexitswithactivegoroutines) | Завершение main и фоновые горутины | [Жизненный цикл горутин](/go/concurrency/goroutine-lifecycle) | Дополнительное чтение |
| [32](https://golang50shades.com/#anameunbuf_ch_send_doneasendingtoanunbufferedchannelreturnsassoonasthetargetreceiverisready) | Момент завершения отправки в канал | [Каналы](/go/concurrency/channels) | Дополнительное чтение |
| [33](https://golang50shades.com/#anameclosed_ch_sendasendingtoanclosedchannelcausesapanic) | Отправка после закрытия канала | [Каналы](/go/concurrency/channels) | Дополнительное чтение |
| [34](https://golang50shades.com/#anameusing_nil_chausingnilchannels) | Нулевой канал в select | [select](/go/concurrency/select) | Дополнительное чтение |
| [35](https://golang50shades.com/#anamemethod_val_receiveramethodswithvaluereceiverscantchangetheoriginalvalue) | Копирование receiver по значению | [Методы и встраивание](/go/api/methods-embedding) | Дополнительное чтение |
| [36](https://golang50shades.com/#anameclose_http_resp_bodyaclosinghttpresponsebody) | Освобождение тела HTTP-ответа | [HTTP](/go/stdlib/net-http) | Дополнительное чтение |
| [37](https://golang50shades.com/#anameclose_http_connaclosinghttpconnections) | Закрытие HTTP-соединений | [HTTP](/go/stdlib/net-http) | Дополнительное чтение |
| [38](https://golang50shades.com/#anamejson_encode_newline_json_encoder_adds_a_newline_character) | Разделитель в JSON Encoder | [JSON](/go/stdlib/json) | Дополнительное чтение; Перевод строки описан для `json.Encoder`, а не для `json.Marshal`. |
| [39](https://golang50shades.com/#anamejson_escape_html_json_package_escapes_special_html_characters) | Экранирование HTML в JSON | [JSON](/go/stdlib/json) | Дополнительное чтение; Оговорка касается HTML escaping в `encoding/json` v1 и его настроек. |
| [40](https://golang50shades.com/#anamejson_numaunmarshallingjsonnumbersintointerfacevalues) | Тип числа после JSON-декодирования | [JSON](/go/stdlib/json) | Дополнительное чтение; Декодер v1 обычно превращает JSON-число в `float64`; см. `UseNumber`. |
| [41](https://golang50shades.com/#anamejson_utf8_strings_json_string_values_will_not_be_ok_with_hex) | Некорректные UTF-8 байты в JSON | [JSON](/go/stdlib/json) | Дополнительное чтение; Сериализация UTF-8 здесь описана для `encoding/json` v1. |
| [42](https://golang50shades.com/#anamecompare_struct_arr_slice_mapacomparingstructsarraysslicesandmaps) | Какие типы сравнимы | [Значения и указатели](/go/language/values-pointers), [срезы](/go/language/slices), [карты](/go/language/maps) | Дополнительное чтение |
| [43](https://golang50shades.com/#anamepanic_recoverarecoveringfromapanic) | Ограничение recover по стеку | [defer, panic и recover](/go/language/defer-panic-recover) | Включено |
| [44](https://golang50shades.com/#anamerange_val_updateaupdatingandreferencingitemvaluesinslicearrayandmaprangeclauses) | Копия элемента при range | [Циклы и range](/go/language/control-range) | Включено |
| [45](https://golang50shades.com/#anameslice_hidden_dataahiddendatainslices) | Общий массив за видимой частью среза | [Срезы](/go/language/slices) | Дополнительное чтение |
| [46](https://golang50shades.com/#anameslice_data_corruptionaslicedatacorruption) | Запись за границу длины | [Срезы](/go/language/slices) | Дополнительное чтение |
| [47](https://golang50shades.com/#anamestale_slicesastaleslices) | Ссылка на уже перемещённый фрагмент | [Срезы](/go/language/slices) | Дополнительное чтение |
| [48](https://golang50shades.com/#anametype_decl_methodsatypedeclarationsandmethods) | Объявленный тип и его методы | [Методы и встраивание](/go/api/methods-embedding) | Дополнительное чтение |
| [49](https://golang50shades.com/#anamedeep_for_breakoutabreakingoutofforswitchandforselectcodeblocks) | Выход из вложенного управляющего блока | [Циклы и range](/go/language/control-range) | Дополнительное чтение |
| [50](https://golang50shades.com/#anameclosure_for_it_varsaiterationvariablesandclosuresinforstatements) | Захват переменной цикла замыканием | [Функции и замыкания](/go/language/functions-closures) | Включено; Семантика объявленных переменных цикла изменилась в Go 1.22; важна версия языка модуля. |
| [51](https://golang50shades.com/#anamedeferred_callsadeferredfunctioncallargumentevaluation) | Снимок аргумента defer | [defer, panic и recover](/go/language/defer-panic-recover) | Включено |
| [52](https://golang50shades.com/#anamedeferred_call_exeadeferredfunctioncallexecution) | Обратный порядок defer | [defer, panic и recover](/go/language/defer-panic-recover) | Включено |
| [53](https://golang50shades.com/#anamefailed_type_assertafailedtypeassertions) | Проверка результата type assertion | [Интерфейсы](/go/api/interfaces) | Дополнительное чтение |
| [54](https://golang50shades.com/#anameblocked_goroutinesablockedgoroutinesandresourceleaks) | Утечка из-за заблокированной горутины | [Жизненный цикл горутин](/go/concurrency/goroutine-lifecycle) | Дополнительное чтение |
| [55](https://golang50shades.com/#anamezero_size_var_same_address_for_different_zero_sized_vars) | Адрес переменной нулевого размера | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [56](https://golang50shades.com/#anameiota_zero_first_use_of_iota) | Область действия iota | [Значения и указатели](/go/language/values-pointers) | Дополнительное чтение |
| [57](https://golang50shades.com/#anameptr_receiver_val_instausingpointerreceivermethodsonvalueinstances) | Pointer-receiver и адресуемое значение | [Методы и встраивание](/go/api/methods-embedding) | Дополнительное чтение |
| [58](https://golang50shades.com/#anamemap_value_field_updateaupdatingmapvaluefields) | Изменение значения в map | [Карты](/go/language/maps) | Включено |
| [59](https://golang50shades.com/#anamenil_in_nil_in_valsanilinterfacesandnilinterfacesvalues) | Интерфейс с нулевым динамическим значением | [Интерфейсы](/go/api/interfaces) | Дополнительное чтение |
| [60](https://golang50shades.com/#anamestack_heap_varsastackandheapvariables) | Escape-поведение и время жизни значения | [Стек и escape analysis](/go/runtime/stacks-escape) | Дополнительное чтение |
| [61](https://golang50shades.com/#anamegomaxprocsagomaxprocsconcurrencyandparallelism) | GOMAXPROCS и доступные CPU | [Планировщик и netpoll](/go/runtime/scheduler-netpoll) | Дополнительное чтение; Default `GOMAXPROCS` учитывает CPU affinity и cgroup quota и может обновляться; сверяйте выпуск Go. |
| [62](https://golang50shades.com/#anamerw_reorderareadandwriteoperationreordering) | Порядок чтений и записей | [Модель памяти и атомики](/go/concurrency/memory-model-atomic) | Дополнительное чтение |
| [63](https://golang50shades.com/#anamepschedapreemptivescheduling) | Вытеснение при планировании | [Планировщик и netpoll](/go/runtime/scheduler-netpoll) | Дополнительное чтение; Это описание реализации планировщика, а не гарантия спецификации; сверяйте версию. |
| [64](https://golang50shades.com/#anamecgo_multiline_import_c_importc_and_multiline) | Многострочный импорт C | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение; Поведение cgo зависит от версии Go, toolchain и платформы. |
| [65](https://golang50shades.com/#anamecgo_noblanks_no_blank_lines_between_import_comments) | Пустые строки в блоке cgo | [Пакеты и API](/go/api/packages-api) | Дополнительное чтение; Правила комментариев импорта C сверяйте с используемым toolchain. |
| [66](https://golang50shades.com/#anamecgo_no_var_args_cant_call_cfunc_with_var_args) | Ограничение variadic-вызова из cgo | [Сборка и toolchain](/go/tooling/toolchain-build) | Дополнительное чтение; Ограничение C variadic-вызовов относится к границе cgo и внешнему ABI. |

Каждая ссылка ведёт к исходному H6-якорю; запись не означает, что весь внешний сайт просмотрен или включён. Текущую семантику сверяйте со [спецификацией языка](https://go.dev/ref/spec), официальными release notes и связанной статьёй справочника.
