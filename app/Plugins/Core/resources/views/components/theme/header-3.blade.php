@extends('plugins.Core.app')
@section('content')
    <header class="text-gray-600 body-font" _msthidden="6">
        <div class="container mx-auto flex flex-wrap p-5 flex-col md:flex-row items-center" _msthidden="6">
            <nav class="flex lg:w-2/5 flex-wrap items-center text-base md:ml-auto" _msthidden="4">
                <a class="mr-5 hover:text-gray-900" _msthash="684831" _msttexthash="132652" _msthidden="1">First Link</a>
                <a class="mr-5 hover:text-gray-900" _msthash="684832" _msttexthash="151060" _msthidden="1">Second Link</a>
                <a class="mr-5 hover:text-gray-900" _msthash="684833" _msttexthash="130351" _msthidden="1">Third Link</a>
                <a class="hover:text-gray-900" _msthash="684834" _msttexthash="154895" _msthidden="1">Fourth Link</a>
            </nav>
            <a class="flex order-first lg:order-none lg:w-1/5 title-font font-medium items-center text-gray-900 lg:items-center lg:justify-center mb-4 md:mb-0" _msthidden="1">
                <svg xmlns="http://www.w3.org/2000/svg" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" class="w-10 h-10 text-white p-2 bg-indigo-500 rounded-full" viewBox="0 0 24 24">
                    <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"></path>
                </svg>
                <span class="ml-3 text-xl" _msthash="739766" _msttexthash="156481" _msthidden="1">CodeFec</span>
            </a>
            <div class="lg:w-2/5 inline-flex lg:justify-end ml-5 lg:ml-0" _msthidden="1">
                <button class="inline-flex items-center bg-gray-100 border-0 py-1 px-3 focus:outline-none hover:bg-gray-200 rounded text-base mt-4 md:mt-0" _msthidden="1">
                    <font _mstmutation="1" _msthash="964613" _msttexthash="79859" _msthidden="1">Button</font>
                    <svg fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" class="w-4 h-4 ml-1" viewBox="0 0 24 24">
                        <path d="M5 12h14M12 5l7 7-7 7"></path>
                    </svg>
                </button>
            </div>
        </div>
    </header>
@endsection