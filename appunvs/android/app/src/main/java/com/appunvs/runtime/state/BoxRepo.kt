// BoxRepo — Compose-observable mirror of the user's box list, backed
// by real /box endpoint calls.  Mirrors iOS BoxStore.
package com.appunvs.runtime.state

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.appunvs.runtime.net.BoxCreateRequest
import com.appunvs.runtime.net.BoxWire
import com.appunvs.runtime.net.RelayApi
import kotlinx.coroutines.launch

class BoxRepo(private val api: RelayApi) : ViewModel() {

    val boxes = mutableStateListOf<BoxWire>()
    var activeBox by mutableStateOf<BoxWire?>(null)
        private set
    var loading by mutableStateOf(false)
        private set
    var lastError by mutableStateOf<String?>(null)

    fun refresh() {
        viewModelScope.launch {
            loading = true
            try {
                val next = api.listBoxes().boxes
                boxes.clear()
                boxes.addAll(next)
                // Preserve active selection by id; default to first if gone.
                activeBox = next.firstOrNull { it.boxID == activeBox?.boxID }
                    ?: next.firstOrNull()
            } catch (t: Throwable) {
                lastError = t.message ?: t.javaClass.simpleName
            } finally {
                loading = false
            }
        }
    }

    fun setActive(box: BoxWire) {
        activeBox = box
    }

    fun create(title: String) {
        viewModelScope.launch {
            try {
                val resp = api.createBox(BoxCreateRequest(title = title))
                boxes.add(0, resp.box)
                activeBox = resp.box
            } catch (t: Throwable) {
                lastError = t.message ?: t.javaClass.simpleName
            }
        }
    }
}
