<ul class="navbar-nav">
    @foreach (Itf()->get('menu') as $key => $value)
        {{-- 必须是父级菜单 --}}
        @if (!arr_has($value, 'parent_id'))
            @if(arr_has($value,"quanxian"))
                @if(auth()->check())
                    @include('App::layouts.themes.menu.menu-quanxian')
                @endif
            @else
                @include('App::layouts.themes.menu.menu')
            @endif
        @endif
    @endforeach
</ul>
