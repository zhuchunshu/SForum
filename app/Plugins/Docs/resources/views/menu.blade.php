<div class="d-none d-lg-block col-lg-3">
    <ul class="nav nav-pills nav-vertical">
        @if(core_Str_menu_url('/' . request()->path()) === '/docs')
            <li class="nav-item">
                <a href="/docs" id="#docs-menu-index" class="nav-link active">
                    Documentation
                </a>
            </li>
        @else
            <li class="nav-item">
                <a href="/docs" id="#docs-menu-index" class="nav-link">
                    Documentation
                </a>
            </li>
        @endif
        @foreach($docsAll as $id=>$data)
            @if(count($data['docs']))
                <li class="nav-item">
                    <a href="#docs-menu-{{$id}}" class="nav-link" data-bs-toggle="collapse" aria-expanded="false">
                        {{$data['name']}}
                        <span class="nav-link-toggle"></span>
                    </a>
                    <ul class="nav nav-pills collapse" id="docs-menu-{{$id}}">
                        @foreach($data['docs'] as $doc)
                            <li class="nav-item">
                                @if(core_Str_menu_url('/' . request()->path()) === '/docs/'.$id.'/'.$doc['id'].".html")
                                    <a href="/docs/{{$id}}/{{$doc['id']}}.html" docs-menu="active" class="nav-link active">
                                        {{$doc['title']}}
                                    </a>
                                @else
                                    <a href="/docs/{{$id}}/{{$doc['id']}}.html" class="nav-link">
                                        {{$doc['title']}}
                                    </a>
                                @endif

                            </li>
                        @endforeach
                    </ul>
                </li>
            @else
                @if(core_Str_menu_url('/' . request()->path()) === '/docs/'.$id)
                    <li class="nav-item">
                        <a href="/docs/{{$id}}" class="nav-link active">
                            {{$data['name']}}
                        </a>
                    </li>
                @else
                    <li class="nav-item">
                        <a href="/docs/{{$id}}" class="nav-link">
                            {{$data['name']}}
                        </a>
                    </li>
                @endif
            @endif
        @endforeach
    </ul>
</div>

