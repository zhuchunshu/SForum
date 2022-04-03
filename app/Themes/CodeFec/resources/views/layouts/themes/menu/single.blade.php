@if (core_Str_menu_url('/' . request()->path()) === $value['url'])
    <li class="nav-item active">
    @else
    <li class="nav-item">
@endif
<a class="nav-link" id="admin-menu-{{ $key }}" href="{{ $value['url'] }}">
    <span class="nav-link-icon d-md-none d-lg-inline-block">
        <!-- Download SVG icon from http://tabler-icons.io/i/package -->
        @if ($value['icon'])
            {!! $value['icon'] !!}
        @else
            <svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-circle" width="24" height="24"
                viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round"
                stroke-linejoin="round">
                <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
                <circle cx="12" cy="12" r="9"></circle>
            </svg>
        @endif
    </span>
    <span class="nav-link-title">
        {{ $value['name'] }}
    </span>
</a>

</li>
