<ul class="navbar-nav">
    @foreach (Itf()->get('menu') as $key => $value)
        {{-- 必须是父级菜单 --}}
        @if (!arr_has($value, 'parent_id'))
            @if(arr_has($value,"quanxian") && auth()->check())
                @include('Core::layouts.themes.menu.menu-quanxian')
            @else
                @include('Core::layouts.themes.menu.menu')
            @endif
        @endif
    @endforeach
</ul>
