@extends('app')

@section('title','Logger')

@section('content')
    <div class="row row-cards">
        <div class="card">
            <div class="card-body">
                <div id="table-default" class="table-responsive">
                    <table class="table">
                        <thead>
                        <tr>
                            <th><button data-sort="sort-id" class="table-sort">Id</button></th>
                            <th><button data-sort="sort-store" class="table-sort">父名</button></th>
                            <th><button data-sort="sort-name" class="table-sort">子名</button></th>
                            <th><button data-sort="sort-log" class="table-sort">日志内容</button></th>
                            <th><button data-sort="sort-date" class="table-sort">创建时间</button></th>
                            <th></th>
                        </tr>
                        </thead>
                        <tbody class="table-tbody">
                            @if(!$page->count())
                                <tr>
                                    <td class="sort-id">无结果</td>
                                    <td class="sort-store">无结果</td>
                                    <td class="sort-name">无结果</td>
                                    <td class="sort-log">无结果</td>
                                    <td class="sort-data">无结果</td>
                                    <td class="sort-data">无结果</td>
                                </tr>
                            @else
                                @foreach($page->items() as $data)
                                    <tr>
                                        <td class="sort-id">{{$data['_id']}}</td>
                                        <td class="sort-store">{{$data['store']}}</td>
                                        <td class="sort-name">{{$data['name']}}</td>
                                        <td class="sort-log">{{$data['log']}}</td>
                                        <td class="sort-date" data-date="{{strtotime($data['created_at'])}}">{{$data['created_at']}}</td>
                                        <td><a href="/admin/server/logger/{{$data['_id']}}.html">查看</a></td>
                                    </tr>
                                @endforeach
                            @endif
                        </tbody>
                    </table>
                </div>
            </div>
            @if($page->count())
                {!! make_page($page) !!}
            @endif
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{file_hash('tabler/libs/list.js/dist/list.min.js')}}" defer></script>
    <script>
        document.addEventListener("DOMContentLoaded", function() {
            const list = new List('table-default', {
                sortClass: 'table-sort',
                listClass: 'table-tbody',
                valueNames: [ 'sort-id', 'sort-name', 'sort-store', 'sort-log',
                    { attr: 'data-date', name: 'sort-date' },
                    // { attr: 'data-progress', name: 'sort-progress' },
                    // 'sort-quantity'
                ]
            });
        })
    </script>
@endsection