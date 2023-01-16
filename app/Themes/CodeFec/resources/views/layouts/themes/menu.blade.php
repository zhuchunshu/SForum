<ul class="navbar-nav">
    @foreach (_menu() as $key => $value)
        {{-- 必须是父级菜单 --}}
        @if (!arr_has($value, 'parent_id') && !$value['hidden'])
            @if(arr_has($value,"quanxian") && $value['quanxian'] instanceof \Closure)
                @include('App::layouts.themes.menu.menu-quanxian')
            @else
                @include('App::layouts.themes.menu.menu')
            @endif
        @endif
    @endforeach
</ul>
