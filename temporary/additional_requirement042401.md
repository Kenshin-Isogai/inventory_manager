

- subscope (scopeの下位区分) が欲しい。Device で開発装置の指定、scope で光学系か, 機械系か, 制御系かなどを指定するとして、少なくともその下位に、どこの部位の光学系か, どの機械assembly か、なども実用上ほぼ必須になりそうなので。
- snapshot とは別に、各製品の増減 flow が確認できると良いと思った。(2026/05/10: +10 (arrival:PO\_number,2026/05/12: -20:reservation), のような感じで)
- snapshot 機能が欲しい。具体的に言うと、日付を設定すると、その日付におけるavailable な在庫の個数(locationが別なら別行扱い) を予約の確保期間や納期などに基づいた在庫の予想増減から算出して一覧として出力する機能。
- 基本的に、items や orders, snapshot など, item\_numberやmanufacturer, quotation\_number, purchase\_order\_number, category などを入力して filtering できるようにしたい。なお、categoryやitem\_number については予測機能やリスト等、すでにDB内に存在している名称をはっきり覚えていなくても検索できるようにしたい。
- requirements, reservation, shotage 画面は分けたほうが良いのでしょうか？どちらかというと、scope, subscope の一覧画面があって、そこに各scope, subscope のreservations, shoetage, requirements の概要情報を載せ、そこから各scope, subscope 毎を指定したら、そのscope, subscopeの詳細情報を確認したり、Reservations のCRUD 操作や shortage 確認などを行えると良いと思った。
- 各scopeには開始日時が指定されると思うが、shortage の計算は、開始日時までに使用可能になりそうな分と、開始日時からdelay して使用可能になる分を分けて表示できるようにしてほしい。後者の場合は、何月何日に何個増えるかなども見れると良いと思った。
- item の description については、後から編集できる機能があってもいいと思った。それと、まだ、reservation や order, quotation などと紐づいていない item については delete できる機能が欲しい。 
- reservation や　各scope, subscope の必要物品, shortage 物品をcsv で出力できるようにしてほしい。
- reservation や　各scope, subscope の必要物品 を csv で追加できるようにしてほしい。
- scope, subscope の必要物品一覧から　直接 reservation したりする機能ってあるのでしたっけ？なければそういう機能が欲しいです。reservation については、reservation開始日付が未来の場合は、現在在庫だけでなく、その日付までにavailable になる物品についても行えるようにしてほしい。(orders を自動で割り当てる。)自動割り当てがあるので、preview 画面は必要として。
- 現在、reservation は order と紐づけられるのでしたっけ？在庫から割り当てるだけではなく、将来のexpected arrival まで考慮するとすると、紐づける必要がある気がします。
- scope,subscope に使えそうな物品が 物品のarrivalなどで在庫に増える場合、日付だけでなく、その充当される部品に紐づいたorder を確認できると良いと思う。
- scope, subscope の必要物品一覧を登録する際に、もしDB内未登録の物品が含まれていた場合、事前にそのitem を登録する操作が必要になるかと思うが、scope, subscope の必要物品一覧を登録する操作とitem を登録する操作 が分離されすぎている(いちいち画面を手動で行き来する必要がある) とUXが悪いと思う。
- UI の各所で item number を表示する場面があるかと思う。品番だけだと分からないものも多いはずなので、items のmanufacturer や description などを、いちいちitems 画面に遷移せずに確認できる手段が欲しい。
- arrival の到着予想をカレンダーで確認できる機能が欲しい。到着物品とその数や、そのexpected arrival date がどのquotation, order に紐づいているかもその画面から手軽に確認できると良いと思った。(その1画面に全て載せたいのではなく、面倒な手動画面遷移を挟まなくても済むという意味。)
- 物品やorders, quotations など、list で一覧を見せる UI　は基本的に collapse や filtering 機能が有ってほしい。


