<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class PluginQqPusherV110Migration extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::dropIfExists('qqpusher');
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('qqpusher', function (Blueprint $table) {
            //
        });
    }
}
